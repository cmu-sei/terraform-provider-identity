## Identity Accounts
### Important Note: This resource type will not work as expected until Staging/Development are updated to the latest version of identity.
The provider can interact with the Identity API to create and manage Identity accounts. The Player Provider can then be used to create Player users corresponding to these accounts.

There are some differences to note between this type and other resource types. Identity accounts cannot be truly deleted. A `terraform destroy` will simply deactivate the targeted account. The only fields that can be truly updated (ie updated without creating anything new) are an account's role and a property's value. The key of a property can be updated, but doing so requires creating a new property. This is handled automatically by the provider.

### Properties
Properties are blocks that can be added to an account. When an account is created, the API will automatically assign it a name, username, and email property. These three properties are ignored by the provider. 

See below for an example of an account and property.

```
resource "crucible_identity_account" "test" {
    username = "someUserName@sei.cmu.edu"
    password = "Password"
    role = "Member"

    property {
      key = "foo"
      value = "bar"
    }
}
```

- username: The username for the account. Note that it must be an email address with a valid domain. Required.
- password: This account's password. Optional. 
- role: This account's role. Optional
- status: Whether this account is active. Computed.
- global_id: This account's GUID. Use this to add a corresponding user to a Player team. Computed.

property
- account_id: The id of the account this property is set on. Computed.
- key: The key for this property. This cannot be changed in-place. If it is changed, a new property will be created with the new key value. Required.
- value: The value for this property. This can be updated normally. Required.

## Reporting bugs and requesting features

Think you found a bug? Please report all Crucible bugs - including bugs for the individual Crucible apps - in the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include as much detail as possible including steps to reproduce, specific app involved, and any error messages you may have received.

Have a good idea for a new feature? Submit all new feature requests through the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include the reasons why you're requesting the new feature and how it might benefit other Crucible users.
