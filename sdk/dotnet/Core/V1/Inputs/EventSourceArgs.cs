// *** WARNING: this file was generated by pulumigen. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Threading.Tasks;
using Pulumi.Serialization;

namespace Pulumi.Kubernetes.Types.Inputs.Core.V1
{

    /// <summary>
    /// EventSource contains information for an event.
    /// </summary>
    public class EventSourceArgs : Pulumi.ResourceArgs
    {
        /// <summary>
        /// Component from which the event is generated.
        /// </summary>
        [Input("component")]
        public Input<string>? Component { get; set; }

        /// <summary>
        /// Node name on which the event is generated.
        /// </summary>
        [Input("host")]
        public Input<string>? Host { get; set; }

        public EventSourceArgs()
        {
        }
    }
}