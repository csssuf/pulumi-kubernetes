// Copyright 2016-2017, Pulumi Corporation.  All rights reserved.

package tfbridge

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	pbempty "github.com/golang/protobuf/ptypes/empty"
	pbstruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pkg/errors"
	"github.com/pulumi/lumi/pkg/resource"
	"github.com/pulumi/lumi/pkg/resource/plugin"
	"github.com/pulumi/lumi/pkg/resource/provider"
	"github.com/pulumi/lumi/pkg/tokens"
	"github.com/pulumi/lumi/pkg/util/contract"
	"github.com/pulumi/lumi/sdk/go/pkg/lumirpc"
	"golang.org/x/net/context"
)

// Provider implements the Lumi resource provider operations for any Terraform plugin.
type Provider struct {
	host      *provider.HostClient       // the RPC link back to the Lumi engine.
	info      ProviderInfo               // overlaid info about this provider.
	tf        terraform.ResourceProvider // the Terraform resource provider to use.
	module    string                     // the Terraform module name.
	resources map[tokens.Type]Resource   // a map of Lumi type tokens to resource info.
}

// Resource wraps both the Terraform resource type info plus the overlay resource info.
type Resource struct {
	TF     terraform.ResourceType // Terraform resource info.
	Schema ResourceInfo           // optional provider overrides.
}

// NewProvider creates a new Lumi RPC server wired up to the given host and wrapping the given Terraform provider.
func NewProvider(host *provider.HostClient, tf terraform.ResourceProvider, module string) *Provider {
	// TODO: audit computed logic to ensure we flow from Lumi's notion of unknowns to TF computeds properly.
	p := &Provider{
		host:   host,
		info:   Providers[module],
		tf:     tf,
		module: module,
	}
	p.initResourceMap()
	return p
}

var _ lumirpc.ResourceProviderServer = (*Provider)(nil)

func (p *Provider) pkg() tokens.Package      { return tokens.Package(p.module) }
func (p *Provider) indexMod() tokens.Module  { return tokens.Module(p.pkg() + ":index") }
func (p *Provider) configMod() tokens.Module { return tokens.Module(p.pkg() + ":config/vars") }

// resource looks up the Terraform resource provider from its Lumi type token.
func (p *Provider) resource(t tokens.Type) (Resource, bool) {
	res, has := p.resources[t]
	return res, has
}

// initResourceMap creates a simple map from Lumi to Terraform resource type.
func (p *Provider) initResourceMap() {
	prefix := p.module + "_" // all resources will have this prefix.

	// Fetch a list of all resource types handled by this provider and make a map.
	p.resources = make(map[tokens.Type]Resource)
	for _, res := range p.tf.Resources() {
		var tok tokens.Type

		// See if there is override information for this resource.  If yes, use that to decode the token.
		var schema ResourceInfo
		if p.info.Resources != nil {
			schema = p.info.Resources[res.Name]
			tok = schema.Tok
		}

		// Otherwise, we default to the standard naming scheme.
		if tok == "" {
			// Strip off the module prefix (e.g., "aws_").
			contract.Assertf(strings.HasPrefix(res.Name, prefix),
				"Expected all Terraform resources in this module to have a '%v' prefix", prefix)
			name := res.Name[len(prefix):]

			// Create a camel name for the module and pascal for the resource type.
			camelName := TerraformToLumiName(name, false)
			pascalName := TerraformToLumiName(name, true)

			// Now just manufacture a token with the package, module, and resource type name.
			tok = tokens.Type(string(p.pkg()) + ":" + camelName + ":" + pascalName)
		}

		p.resources[tok] = Resource{TF: res, Schema: schema}
	}
}

// Some functions used below for name and value transformations.
var (
	// terraformKeyRepl swaps out Terraform names for Lumi names.
	terraformKeyRepl = func(k string) (resource.PropertyKey, bool) {
		// TODO: we need to respect the schema maps.
		return resource.PropertyKey(TerraformToLumiName(k, false)), true
	}
	// terraformValueRepl does the reverse, and swaps out Terraform ints for Lumi float64s.
	terraformValueRepl = func(v interface{}) (resource.PropertyValue, bool) {
		if i, isint := v.(int); isint {
			return resource.NewNumberProperty(float64(i)), true
		}
		return resource.PropertyValue{}, false
	}
)

// terraformToLumiProps expands a Terraform-style flatmap into an expanded Lumi resource property map.
func (p *Provider) terraformToLumiProps(props map[string]string) resource.PropertyMap {
	res := make(map[string]interface{})
	for _, key := range flatmap.Map(props).Keys() {
		res[key] = flatmap.Expand(props, key)
	}
	return resource.NewPropertyMapFromMapRepl(res, terraformKeyRepl, terraformValueRepl)
}

// getInfoFromLumiName does a reverse map lookup to find the Terraform name and schema info for a Lumi name, if any.
func (p *Provider) getInfoFromLumiName(key resource.PropertyKey, schema map[string]SchemaInfo) (string, SchemaInfo) {
	// To do this, we will first look to see if there's a known custom schema that uses this name.  If yes, we
	// prefer to use that.  To do this, we must use a reverse lookup.  (In the future we may want to make a
	// lookaside map to avoid the traversal of this map.)  Otherwise, use the standard name mangling scheme.
	ks := string(key)
	for tfname, schinfo := range schema {
		if schinfo.Name == ks {
			return tfname, schinfo
		}
	}
	return LumiToTerraformName(ks), schema[ks]
}

// createTerraformInputs takes a property map plus custom schema info and does whatever is necessary to prepare it for
// use by Terraform.  Note that this function may have side effects, for instance if it is necessary to spill an asset
// to disk in order to create a name out of it.  Please take care not to call it superfluously!
func (p *Provider) createTerraformInputs(m resource.PropertyMap,
	schema map[string]SchemaInfo) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Enumerate the inputs provided and add them to the map using their Terraform names.
	for key, value := range m {
		// Skip any special properties.
		k := string(key)
		if IsBuiltinLumiProperty(k) {
			continue
		}

		// First translate the Lumi property name to a Terraform name.
		name, info := p.getInfoFromLumiName(key, schema)
		contract.Assert(name != "")

		// And then translate the property value.
		v, err := p.createTerraformInput(name, value, info)
		if err != nil {
			return nil, err
		}
		result[name] = v
	}

	// Now enumerate and propagate defaults if the corresponding values are still missing.
	for key, info := range schema {
		if _, has := result[key]; !has {
			if info.Default.Value != nil {
				result[key] = info.Default.Value
			} else if from := info.Default.From; from != "" {
				fk := resource.PropertyKey(from)
				if fromv, hasfrom := m[fk]; hasfrom {
					// Create a Terraform name so we can recover the transformed value and use it.
					tfname, info := p.getInfoFromLumiName(fk, schema)
					v, err := p.createTerraformInput(tfname, fromv, info)
					if err != nil {
						return nil, err
					}
					if info.Default.FromTransform != nil {
						v = info.Default.FromTransform(v)
					}
					result[key] = v
				}
			}
		}
	}

	return result, nil
}

// createTerraformInput takes a single property plus custom schema info and does whatever is necessary to prepare it for
// use by Terraform.  Note that this function may have side effects, for instance if it is necessary to spill an asset
// to disk in order to create a name out of it.  Please take care not to call it superfluously!
func (p *Provider) createTerraformInput(name string,
	v resource.PropertyValue, schema SchemaInfo) (interface{}, error) {
	if v.IsNull() {
		return nil, nil
	} else if v.IsBool() {
		return v.BoolValue(), nil
	} else if v.IsNumber() {
		return int(v.NumberValue()), nil // convert floats to ints.
	} else if v.IsString() {
		return v.StringValue(), nil
	} else if v.IsArray() {
		// FIXME: marshal/unmarshal sets properly.
		var arr []interface{}
		for i, elem := range v.ArrayValue() {
			var eleminfo SchemaInfo
			if schema.Elem != nil {
				eleminfo = *schema.Elem
			}
			e, err := p.createTerraformInput(fmt.Sprintf("%v[%v]", name, i), elem, eleminfo)
			if err != nil {
				return nil, err
			}
			arr = append(arr, e)
		}
		return arr, nil
	} else if v.IsAsset() {
		// We require that there be asset information, otherwise an error occurs.
		if schema.Asset == nil {
			return nil,
				errors.Errorf("Encountered an asset %v but asset translation instructions were missing", name)
		} else if !schema.Asset.IsAsset() {
			return nil,
				errors.Errorf("Invalid asset translation instructions for %v; expected an asset", name)
		}
		return schema.Asset.TranslateAsset(v.AssetValue())
	} else if v.IsArchive() {
		// We require that there be archive information, otherwise an error occurs.
		if schema.Asset == nil {
			return nil,
				errors.Errorf("Encountered an archive %v but asset translation instructions were missing", name)
		} else if !schema.Asset.IsArchive() {
			return nil,
				errors.Errorf("Invalid asset translation instructions for %v; expected an archive", name)
		}
		return schema.Asset.TranslateArchive(v.ArchiveValue())
	}
	contract.Assert(v.IsObject())
	return p.createTerraformInputs(v.ObjectValue(), schema.Fields)
}

// makeTerraformConfig creates a Terraform config map, used in state and diff calculations, from a Lumi property map.
func (p *Provider) makeTerraformConfig(m resource.PropertyMap,
	schema map[string]SchemaInfo) (*terraform.ResourceConfig, error) {
	// Convert the resource bag into an untyped map, and then create the resource config object.
	inputs, err := p.createTerraformInputs(m, schema)
	if err != nil {
		return nil, err
	}
	cfg, err := config.NewRawConfig(inputs)
	if err != nil {
		return nil, err
	}
	return terraform.NewResourceConfig(cfg), nil
}

// makeTerraformConfigFromRPC creates a Terraform config map from a Lumi RPC property map.
func (p *Provider) makeTerraformConfigFromRPC(m *pbstruct.Struct,
	schema map[string]SchemaInfo) (*terraform.ResourceConfig, error) {
	props := plugin.UnmarshalProperties(m, plugin.MarshalOptions{SkipNulls: true})
	return p.makeTerraformConfig(props, schema)
}

// makeTerraformPropertyMap converts a Lumi property bag into its Terraform equivalent.  This requires
// flattening everything and serializing individual properties as strings.  This is a little awkward, but it's how
// Terraform represents resource properties (schemas are simply sugar on top).
func (p *Provider) makeTerraformPropertyMap(m resource.PropertyMap,
	schema map[string]SchemaInfo) (map[string]string, error) {
	// Turn the resource properties into a map.  For the most part, this is a straight Mappable, but we use MapReplace
	// because we use float64s and Terraform uses ints, to represent numbers.
	inputs, err := p.createTerraformInputs(m, schema)
	if err != nil {
		return nil, err
	}
	return flatmap.Flatten(inputs), nil
}

// makeTerraformPropertyMapFromRPC unmarshals an RPC property map and calls through to makeTerraformPropertyMap.
func (p *Provider) makeTerraformPropertyMapFromRPC(m *pbstruct.Struct,
	schema map[string]SchemaInfo) (map[string]string, error) {
	props := plugin.UnmarshalProperties(m, plugin.MarshalOptions{SkipNulls: true})
	return p.makeTerraformPropertyMap(props, schema)
}

// makeTerraformDiff takes a bag of old and new properties, and returns two things: the attribute state to use for the
// current resource alongside a Terraform diff for the old and new.  If there was no old state, the first return is nil.
func (p *Provider) makeTerraformDiff(old resource.PropertyMap, new resource.PropertyMap,
	schema map[string]SchemaInfo) (map[string]string, *terraform.InstanceDiff, error) {
	attrs := make(map[string]string)
	diff := make(map[string]*terraform.ResourceAttrDiff)
	// Add all new property values.
	if new != nil {
		// FIXME: avoid spilling except for during creation.
		inputs, err := p.makeTerraformPropertyMap(new, schema)
		if err != nil {
			return nil, nil, err
		}
		for p, v := range inputs {
			if diff[p] == nil {
				diff[p] = &terraform.ResourceAttrDiff{}
			}
			attrs[p] = v
			diff[p].New = v
		}
	}
	// Now add all old property values, provided they exist in new.
	if old != nil {
		// FIXME: avoid spilling except for during creation.  I think maybe we just skip olds or when new==old?
		inputs, err := p.makeTerraformPropertyMap(old, schema)
		if err != nil {
			return nil, nil, err
		}
		for p, v := range inputs {
			if diff[p] != nil {
				diff[p].Old = v
			}
		}
	}
	return attrs, &terraform.InstanceDiff{Attributes: diff}, nil
}

// makeTerraformDiffFromRPC takes RPC maps of old and new properties, unmarshals them, and calls into makeTerraformDiff.
func (p *Provider) makeTerraformDiffFromRPC(old *pbstruct.Struct, new *pbstruct.Struct,
	schema map[string]SchemaInfo) (map[string]string, *terraform.InstanceDiff, error) {
	oldprops := plugin.UnmarshalProperties(old, plugin.MarshalOptions{SkipNulls: true})
	newprops := plugin.UnmarshalProperties(new, plugin.MarshalOptions{SkipNulls: true})
	return p.makeTerraformDiff(oldprops, newprops, schema)
}

// Configure configures the underlying Terraform provider with the live Lumi variable state.
func (p *Provider) Configure() error {
	// Read all properties from the config module.
	props, err := p.host.ReadLocations(tokens.Token(p.configMod()), true)
	if err != nil {
		return errors.Errorf("Error reading config state: %v", err)
	}
	glog.V(9).Infof("Configuring Terraform provider %v: %v var(s) found from '%v'",
		p.module, len(props), p.configMod())

	// Now make a map of each of the config token values.
	config, err := p.makeTerraformConfig(props, p.info.Config)
	if err != nil {
		return errors.Errorf("Error marshaling config state to Terraform: %v", err)
	}

	// Perform validation of the config state so we can offer nice errors.
	keys, errs := p.tf.Validate(config)
	if len(keys) > 0 {
		// TODO: unify this with check by adding a Configure RPC method to the gRPC interface.
		return errors.Errorf("One or more errors occurred while configuring: %v (%v)", keys[0], errs[0])
	}

	// Now actually attempt to do the configuring and return its resulting error (if any).
	return p.tf.Configure(config)
}

// Check validates that the given property bag is valid for a resource of the given type.
func (p *Provider) Check(ctx context.Context, req *lumirpc.CheckRequest) (*lumirpc.CheckResponse, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Check): %v", t)
	}

	// Manufacture a resource config to check, check it, and return any failures that result.
	rescfg, err := p.makeTerraformConfigFromRPC(req.GetProperties(), res.Schema.Fields)
	if err != nil {
		return nil, errors.Errorf("Error preparing %v's property state: %v", t, err)
	}
	keys, errs := p.tf.ValidateResource(res.TF.Name, rescfg)
	var failures []*lumirpc.CheckFailure
	for i, key := range keys {
		failures = append(failures, &lumirpc.CheckFailure{
			Property: key,
			Reason:   errs[i].Error(),
		})
	}
	return &lumirpc.CheckResponse{Failures: failures}, nil
}

// Name names a given resource.  Sometimes this will be assigned by a developer, and so the provider
// simply fetches it from the property bag; other times, the provider will assign this based on its own algorithm.
// In any case, resources with the same name must be safe to use interchangeably with one another.
func (p *Provider) Name(ctx context.Context, req *lumirpc.NameRequest) (*lumirpc.NameResponse, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Name): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.Name: lumi='%v', tf=%v", t, res.TF.Name)

	// All Terraform bridge providers will have a name property that we use for URN naming purposes.
	props := plugin.UnmarshalProperties(req.GetProperties(), plugin.MarshalOptions{SkipNulls: true})
	name, has := props[NameProperty]
	if !has {
		return nil, errors.Errorf("Missing a '%v' property", NameProperty)
	} else if !name.IsString() {
		return nil, errors.Errorf("Expected a string '%v' property; got %v", NameProperty, name)
	}
	namestr := name.StringValue()
	if namestr == "" {
		if req.GetUnknowns()[NameProperty] {
			return nil, errors.Errorf("The '%v' property cannot be a computed expression", NameProperty)
		}
		return nil, errors.Errorf("The '%v' property cannot be the empty string", NameProperty)
	}
	return &lumirpc.NameResponse{Name: namestr}, nil
}

// Create allocates a new instance of the provided resource and returns its unique ID afterwards.  (The input ID
// must be blank.)  If this call fails, the resource must not have been created (i.e., it is "transacational").
func (p *Provider) Create(ctx context.Context, req *lumirpc.CreateRequest) (*lumirpc.CreateResponse, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Create): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.Create: lumi='%v', tf=%v", t, res.TF.Name)

	// Create a new state, with no diff, that is missing an ID.  Terraform will interpret this as a create operation.
	info := &terraform.InstanceInfo{Type: res.TF.Name}
	inputs, err := p.makeTerraformPropertyMapFromRPC(req.GetProperties(), res.Schema.Fields)
	if err != nil {
		return nil, errors.Errorf("Error preparing %v's property state: %v", t, err)
	}
	state := &terraform.InstanceState{Attributes: inputs}
	newstate, err := p.tf.Apply(info, state, &terraform.InstanceDiff{})
	if err != nil {
		return nil, err
	}
	return &lumirpc.CreateResponse{Id: newstate.ID}, nil
}

// Get reads the instance state identified by ID, returning a populated resource object, or an error if not found.
func (p *Provider) Get(ctx context.Context, req *lumirpc.GetRequest) (*lumirpc.GetResponse, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Get): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.Get: lumi='%v', tf=%v", t, res.TF.Name)

	// To read the instance state, create a blank bit of data and ask the resource provider to recompute it.
	info := &terraform.InstanceInfo{Type: res.TF.Name}
	state := &terraform.InstanceState{ID: req.GetId()}
	getstate, err := p.tf.Refresh(info, state)
	if err != nil {
		return nil, errors.Errorf("Error reading %v's state: %v", t, err)
	}
	props := p.terraformToLumiProps(getstate.Attributes)
	return &lumirpc.GetResponse{
		Properties: plugin.MarshalProperties(props, plugin.MarshalOptions{}),
	}, nil
}

// InspectChange checks what impacts a hypothetical update will have on the resource's properties.
func (p *Provider) InspectChange(
	ctx context.Context, req *lumirpc.InspectChangeRequest) (*lumirpc.InspectChangeResponse, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (InspectChange): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.InspectChange: lumi='%v', tf=%v", t, res.TF.Name)

	// To figure out if we have a replacement, perform the diff and then look for RequiresNew flags.
	inputs, err := p.makeTerraformPropertyMapFromRPC(req.GetOlds(), res.Schema.Fields)
	if err != nil {
		return nil, errors.Errorf("Error preparing %v old property state: %v", t, err)
	}
	info := &terraform.InstanceInfo{Type: res.TF.Name}
	state := &terraform.InstanceState{
		ID:         req.GetId(),
		Attributes: inputs,
	}
	config, err := p.makeTerraformConfigFromRPC(req.GetNews(), res.Schema.Fields)
	if err != nil {
		return nil, errors.Errorf("Error preparing %v property state: %v", t, err)
	}
	diff, err := p.tf.Diff(info, state, config)
	if err != nil {
		return nil, errors.Errorf("Error diffing %v old and new state: %v", t, err)
	}

	// Each RequiresNew translates into a replacement.
	var replaces []string
	for k, attr := range diff.Attributes {
		if attr.RequiresNew {
			replaces = append(replaces, k)
		}
	}

	return &lumirpc.InspectChangeResponse{Replaces: replaces}, nil
}

// Update updates an existing resource with new values.  Only those values in the provided property bag are updated
// to new values.  The resource ID is returned and may be different if the resource had to be recreated.
func (p *Provider) Update(ctx context.Context, req *lumirpc.UpdateRequest) (*pbempty.Empty, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Delete): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.Update: lumi='%v', tf=%v", t, res.TF.Name)

	// Create a state state with the ID to update, a diff with old and new states, and perform the apply.
	info := &terraform.InstanceInfo{Type: res.TF.Name}
	attrs, diff, err := p.makeTerraformDiffFromRPC(req.GetOlds(), req.GetNews(), res.Schema.Fields)
	if err != nil {
		return nil, errors.Errorf("Error preparing %v old and new state/diffs", t, err)
	}
	state := &terraform.InstanceState{
		ID:         req.GetId(),
		Attributes: attrs,
	}
	if _, err := p.tf.Apply(info, state, diff); err != nil {
		return nil, errors.Errorf("Error applying %v update: %v", t, err)
	}
	return &pbempty.Empty{}, nil
}

// Delete tears down an existing resource with the given ID.  If it fails, the resource is assumed to still exist.
func (p *Provider) Delete(ctx context.Context, req *lumirpc.DeleteRequest) (*pbempty.Empty, error) {
	t := tokens.Type(req.GetType())
	res, has := p.resource(t)
	if !has {
		return nil, fmt.Errorf("Unrecognized resource type (Delete): %v", t)
	}
	glog.V(9).Infof("tfbridge/Provider.Delete: lumi='%v', tf=%v", t, res.TF.Name)

	// Create a new state, with no diff, that is missing an ID.  Terraform will interpret this as a create operation.
	info := &terraform.InstanceInfo{Type: res.TF.Name}
	state := &terraform.InstanceState{ID: req.GetId()}
	if _, err := p.tf.Apply(info, state, &terraform.InstanceDiff{Destroy: true}); err != nil {
		return nil, errors.Errorf("Error apply %v deletion: %v", t, err)
	}
	return &pbempty.Empty{}, nil
}
