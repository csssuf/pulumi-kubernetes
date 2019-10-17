# *** WARNING: this file was generated by the Pulumi Kubernetes codegen tool. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

import warnings
from typing import Optional

import pulumi
import pulumi.runtime
from pulumi import Input, ResourceOptions

from ... import tables, version


class ControllerRevision(pulumi.CustomResource):
    """
    DEPRECATED - apps/v1beta2/ControllerRevision is not supported by Kubernetes 1.16+ clusters. Use
    apps/v1/ControllerRevision instead.
    
    ControllerRevision implements an immutable snapshot of state data. Clients are responsible for
    serializing and deserializing the objects that contain their internal state. Once a
    ControllerRevision has been successfully created, it can not be updated. The API Server will
    fail validation of all requests that attempt to mutate the Data field. ControllerRevisions may,
    however, be deleted. Note that, due to its use by both the DaemonSet and StatefulSet controllers
    for update and rollback, this object is beta. However, it may be subject to name and
    representation changes in future releases, and clients should not depend on its stability. It is
    primarily for internal use by controllers.
    """

    apiVersion: pulumi.Output[str]
    """
    APIVersion defines the versioned schema of this representation of an object. Servers should
    convert recognized schemas to the latest internal value, and may reject unrecognized values.
    More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources
    """

    kind: pulumi.Output[str]
    """
    Kind is a string value representing the REST resource this object represents. Servers may infer
    this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More
    info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
    """

    data: pulumi.Output[dict]
    """
    Data is the serialized representation of the state.
    """

    metadata: pulumi.Output[dict]
    """
    Standard object's metadata. More info:
    https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
    """

    revision: pulumi.Output[int]
    """
    Revision indicates the revision of the state represented by Data.
    """

    def __init__(self, resource_name, opts=None, revision=None, data=None, metadata=None, __name__=None, __opts__=None):
        """
        Create a ControllerRevision resource with the given unique name, arguments, and options.

        :param str resource_name: The _unique_ name of the resource.
        :param pulumi.ResourceOptions opts: A bag of options that control this resource's behavior.
        :param pulumi.Input[int] revision: Revision indicates the revision of the state represented by Data.
        :param pulumi.Input[dict] data: Data is the serialized representation of the state.
        :param pulumi.Input[dict] metadata: Standard object's metadata. More info:
               https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
        """
        if __name__ is not None:
            warnings.warn("explicit use of __name__ is deprecated", DeprecationWarning)
            resource_name = __name__
        if __opts__ is not None:
            warnings.warn("explicit use of __opts__ is deprecated, use 'opts' instead", DeprecationWarning)
            opts = __opts__
        if not resource_name:
            raise TypeError('Missing resource name argument (for URN creation)')
        if not isinstance(resource_name, str):
            raise TypeError('Expected resource name to be a string')
        if opts and not isinstance(opts, pulumi.ResourceOptions):
            raise TypeError('Expected resource options to be a ResourceOptions instance')

        __props__ = dict()

        __props__['apiVersion'] = 'apps/v1beta2'
        __props__['kind'] = 'ControllerRevision'
        if revision is None:
            raise TypeError('Missing required property revision')
        __props__['revision'] = revision
        __props__['data'] = data
        __props__['metadata'] = metadata

        __props__['status'] = None

        opts = pulumi.ResourceOptions.merge(opts, pulumi.ResourceOptions(
            version=version.get_version(),
        ))

        super(ControllerRevision, self).__init__(
            "kubernetes:apps/v1beta2:ControllerRevision",
            resource_name,
            __props__,
            opts)

    @staticmethod
    def get(resource_name, id, opts=None):
        """
        Get the state of an existing `ControllerRevision` resource, as identified by `id`.
        The ID is of the form `[namespace]/[name]`; if `[namespace]` is omitted,
        then (per Kubernetes convention) the ID becomes `default/[name]`.

        Pulumi will keep track of this resource using `resource_name` as the Pulumi ID.

        :param str resource_name: _Unique_ name used to register this resource with Pulumi.
        :param pulumi.Input[str] id: An ID for the Kubernetes resource to retrieve.
               Takes the form `[namespace]/[name]` or `[name]`.
        :param Optional[pulumi.ResourceOptions] opts: A bag of options that control this
               resource's behavior.
        """
        opts = ResourceOptions.merge(opts, pulumi.ResourceOptions(id=id))
        return ControllerRevision(resource_name, opts)

    def translate_output_property(self, prop: str) -> str:
        return tables._CASING_FORWARD_TABLE.get(prop) or prop

    def translate_input_property(self, prop: str) -> str:
        return tables._CASING_BACKWARD_TABLE.get(prop) or prop
