// *** WARNING: this file was generated by pulumigen. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Threading.Tasks;
using Pulumi.Serialization;

namespace Pulumi.Kubernetes.Types.Outputs.Core.V1
{

    [OutputType]
    public sealed class DownwardAPIProjection
    {
        /// <summary>
        /// Items is a list of DownwardAPIVolume file
        /// </summary>
        public readonly ImmutableArray<Pulumi.Kubernetes.Types.Outputs.Core.V1.DownwardAPIVolumeFile> Items;

        [OutputConstructor]
        private DownwardAPIProjection(ImmutableArray<Pulumi.Kubernetes.Types.Outputs.Core.V1.DownwardAPIVolumeFile> items)
        {
            Items = items;
        }
    }
}