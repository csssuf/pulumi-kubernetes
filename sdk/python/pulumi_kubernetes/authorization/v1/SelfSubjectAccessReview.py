import pulumi
import pulumi.runtime

from ... import tables

class SelfSubjectAccessReview(pulumi.CustomResource):
    """
    SelfSubjectAccessReview checks whether or the current user can perform an action.  Not filling
    in a spec.namespace means "in all namespaces".  Self is a special case, because users should
    always be able to check whether they can perform an action
    """
    def __init__(self, __name__, __opts__=None, metadata=None, spec=None, status=None):
        if not __name__:
            raise TypeError('Missing resource name argument (for URN creation)')
        if not isinstance(__name__, str):
            raise TypeError('Expected resource name to be a string')
        if __opts__ and not isinstance(__opts__, pulumi.ResourceOptions):
            raise TypeError('Expected resource options to be a ResourceOptions instance')

        __props__ = dict()

        __props__['apiVersion'] = 'authorization.k8s.io/v1'
        __props__['kind'] = 'SelfSubjectAccessReview'
        if spec is None:
            raise TypeError('Missing required property spec')
        __props__['spec'] = spec
        __props__['metadata'] = metadata
        __props__['status'] = status

        super(SelfSubjectAccessReview, self).__init__(
            "kubernetes:authorization.k8s.io/v1:SelfSubjectAccessReview",
            __name__,
            __props__,
            __opts__)

    def translate_output_property(self, prop: str) -> str:
        return tables._CASING_FORWARD_TABLE.get(prop) or prop

    def translate_input_property(self, prop: str) -> str:
        return tables._CASING_BACKWARD_TABLE.get(prop) or prop
