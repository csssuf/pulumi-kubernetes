// *** WARNING: this file was generated by the Pulumi Kubernetes codegen tool. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import { core } from "../..";
import * as inputs from "../../types/input";
import * as outputs from "../../types/output";
import { getVersion } from "../../version";

    /**
     * Job represents the configuration of a single job.
     * 
     * This resource waits until its status is ready before registering success
     * for create/update, and populating output properties from the current state of the resource.
     * The following conditions are used to determine whether the resource creation has
     * succeeded or failed:
     * 
     * 1. The Job's '.status.startTime' is set, which indicates that the Job has started running.
     * 2. The Job's '.status.conditions' has a status of type 'Complete', and a 'status' set
     *    to 'True'.
     * 3. The Job's '.status.conditions' do not have a status of type 'Failed', with a
     * 	'status' set to 'True'. If this condition is set, we should fail the Job immediately.
     * 
     * If the Job has not reached a Ready state after 10 minutes, it will
     * time out and mark the resource update as Failed. You can override the default timeout value
     * by setting the 'customTimeouts' option on the resource.
     */
    export class Job extends pulumi.CustomResource {
      /**
       * APIVersion defines the versioned schema of this representation of an object. Servers should
       * convert recognized schemas to the latest internal value, and may reject unrecognized
       * values. More info:
       * https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
       */
      public readonly apiVersion: pulumi.Output<"batch/v1">;

      /**
       * Kind is a string value representing the REST resource this object represents. Servers may
       * infer this from the endpoint the client submits requests to. Cannot be updated. In
       * CamelCase. More info:
       * https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
       */
      public readonly kind: pulumi.Output<"Job">;

      /**
       * Standard object's metadata. More info:
       * https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
       */
      public readonly metadata: pulumi.Output<outputs.meta.v1.ObjectMeta>;

      /**
       * Specification of the desired behavior of a job. More info:
       * https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
       */
      public readonly spec: pulumi.Output<outputs.batch.v1.JobSpec>;

      /**
       * Current status of a job. More info:
       * https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
       */
      public readonly status: pulumi.Output<outputs.batch.v1.JobStatus>;

      /**
       * Get the state of an existing `Job` resource, as identified by `id`.
       * The ID is of the form `[namespace]/<name>`; if `namespace` is omitted, then (per
       * Kubernetes convention) the ID becomes `default/<name>`.
       *
       * Pulumi will keep track of this resource using `name` as the Pulumi ID.
       *
       * @param name _Unique_ name used to register this resource with Pulumi.
       * @param id An ID for the Kubernetes resource to retrieve. Takes the form `[namespace]/<name>`.
       * @param opts Uniquely specifies a CustomResource to select.
       */
      public static get(name: string, id: pulumi.Input<pulumi.ID>, opts?: pulumi.CustomResourceOptions): Job {
          return new Job(name, undefined, { ...opts, id: id });
      }

      /** @internal */
      private static readonly __pulumiType = "kubernetes:batch/v1:Job";

      /**
       * Returns true if the given object is an instance of Job.  This is designed to work even
       * when multiple copies of the Pulumi SDK have been loaded into the same process.
       */
      public static isInstance(obj: any): obj is Job {
          if (obj === undefined || obj === null) {
              return false;
          }

          return obj["__pulumiType"] === Job.__pulumiType;
      }

      /**
       * Create a batch.v1.Job resource with the given unique name, arguments, and options.
       *
       * @param name The _unique_ name of the resource.
       * @param args The arguments to use to populate this resource's properties.
       * @param opts A bag of options that control this resource's behavior.
       */
      constructor(name: string, args?: inputs.batch.v1.Job, opts?: pulumi.CustomResourceOptions) {
          const props: pulumi.Inputs = {};

          props["apiVersion"] = "batch/v1";
          props["kind"] = "Job";
          props["metadata"] = args && args.metadata || undefined;
          props["spec"] = args && args.spec || undefined;

          props["status"] = undefined;

          if (!opts) {
              opts = {};
          }

          if (!opts.version) {
              opts.version = getVersion();
          }

          super(Job.__pulumiType, name, props, opts);
      }
    }
