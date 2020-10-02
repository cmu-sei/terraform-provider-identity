## Identity Accounts
### Important Note: This resource type will not work as expected until Staging/Development are updated to the latest version of identity.
The provider can interact with the Identity API to create and manage Identity accounts. Whenever an identity account is created, the provider will automatically create an corresponding Player user. This allows users of the provider to create users and add them to teams seamlessly within the same configuration. 

There are some differences to note between this type and other resource type. Identity accounts cannot be truly deleted. A `terraform destroy` will simply deactivate the targeted account. The only fields that can be truly updated (ie updated without creating anything new) are an account's role and a property's value. The key of a property can be updated, but doing so requires creating a new property. This is handled automatically by the provider.

### Proprties
Properties are blocks that can be added to an account. When an account is created, the API will automatically assign it a name, username, and email property. These three properties are ignored by the provider. 

See below for an example of an account and property.

```
resource "crucible_identity_account" "test" {
    username = "providerdevtest@sei.cmu.edu"
    password = "Tartans@1"
    role = "Member"

    property {
      key = "foo"
      value = "bar"
    }
}
```
<ul>
<li> username: The username for the account. Note that it must be an email address with a valid domain. When the Player user corresponding to this account is created, everything in username before the @ will be used as that user's name. Required.
<li> password: This account's password. Optional. 
<li> role: This account's role. Optional
<li> status: Whether this account is active. Computed.
<li> global_id: This account's GUID. Use this to add a corresponding user to a Player team. Computed.
<li> property <ul>
<li> account_id: The id of the account this property is set on. Computed.
<li> key: The key for this property. This cannot be changed in-place. If it is changed, a new property will be created with the new key value. Required.
<li> value: The value for this property. This can be updated normally. Required.
<ul>
<ul>