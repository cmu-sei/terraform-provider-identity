# Terraform Provider Identity Readme

## Identity Provider

The following environment variables need to be set:
```
SEI_IDENTITY_USERNAME=<your username>
SEI_IDENTITY_PASSWORD=<your password>
SEI_IDENTITY_TOK_URL=<the url where you get your identity auth token>
SEI_IDENTITY_CLIENT_ID=<your client ID for authenticating with the Identity API>
SEI_IDENTITY_CLIENT_SECRET=<your client secret for authentication>
SEI_IDENTITY_API_URL=<the url of the identity api>
```

## Identity Accounts

The provider can interact with the Identity API to create and manage Identity accounts. The Player Provider can then be used to create Player users corresponding to these accounts.

There are some differences to note between this type and other resource types. Identity accounts cannot be truly deleted. A `terraform destroy` will simply deactivate the targeted account. The only fields that can be truly updated (that is, updated without creating anything new) are an account's role and a property's value. The key of a property can be updated, but doing so requires creating a new property. This is handled automatically by the provider.

## Properties

Properties are blocks that can be added to an account. When an account is created, the API will automatically assign it a name, username, and email property. These three properties are ignored by the provider. 

See below for an example of an `account` and `property`. Note that some account fields are not shown in the example because they are computed by the provider. See below for details on the fields. 

```
resource "identity_account" "Demo" {
    username = "someUserName@sei.cmu.edu"
    password = "Password"
    role = "Member"

    property {
      key = "foo"
      value = "bar"
    }
}
```

### Top-level account fields

- **username:** The username for the account. Note that it must be an email address with a valid domain. *Required*.
- **password:** This account's password. *Optional*. 
- **role:** This account's role. *Optional*.
- **status:** Whether this account is active. *Computed*.
- **global_id:** This account's GUID. Use this to add a corresponding user to a Player team. *Computed*.

### Property fields

- **account_id:** The id of the account this property is set on. *Computed*.
- **key:** The key for this property. This cannot be changed in-place. If it is changed, a new property will be created with the new key value. *Required*.
- **value:** The value for this property. This can be updated normally. *Required*.

## Identity Clients

The provider can also be used to create Identity clients. Unlike accounts, these can actually be destroyed, so their behavior is in line with a typical Terraform resource type. See below for an example of a client and details on its fields. All optional fields are shown in the example, but computed fields are omitted.

```
resource "identity_client" "Demo" {
  name   = "Demo Client"
  display_name = "Demo Client Display"
  enabled = true
  scopes = "identity-api"
  grants = "client_credentials"
  url {
    type  = "redirectUri"
    value = "http://example.com"
    deleted = false
  }
  url {
    type  = "corsUri"
    value = "http://example.com"
    deleted = false
  }
  url {
    type  = "postLogoutRedirectUri"
    value = "http://example.com"
    deleted = false
  }
  claim {
    value = "something"
    deleted = false
  }
  secret {
    deleted = false
  }
}
```

### Top-level client fields

- **name:** The name for this client. *Required*.
- **display_name:** The name to display for this client. This can be distinct from the name field if desired. *Optional*.
- **enabled:** Whether this client is enabled and can thus be used. *Optional*. Default = `true`.
- **scopes:** The scopes to allow this client to access. *Required*.
- **grants:** The grant types to provide this client. *Optional*. Default = `"client_credentials"`.

### URLs

The client needs one each of a `redirectUri`, `corsUri`, and `postLogoutRedirectUri`. These are specified by the `type` field as shown in the above example.
- **id:** The id of the url. *Computed*.
- **type:** One of the three possible URL types detailed above. *Required*.
- **value:** The actual URL where this URL object should point. *Required*.
- **client_id:** The id of the client this URL is attached to. *Computed*.
- **deleted:** Whether this URL should be deleted. Optional. Default = `false`.

### Claims (optional)

- **id:** The id of this claim. *Computed*.
- **value:** The value for the claim. *Required*.
- **client_id:** The id of the client that this claim is attached to. *Computed*.
- **deleted:** Whether this claim should be deleted. *Optional*. Default = `false`.

### Secrets (optional)

- **id:** The id of the secret. *Computed*.
- **value:** The secret value. *Computed*.
- **deleted:** Whether this secret should be deleted. *Optional*. Default = `false`.

## Reporting bugs and requesting features

Think you found a bug? Please report all Crucible bugs - including bugs for the individual Crucible apps - in the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include as much detail as possible including steps to reproduce, specific app involved, and any error messages you may have received.

Have a good idea for a new feature? Submit all new feature requests through the [cmu-sei/crucible issue tracker](https://github.com/cmu-sei/crucible/issues). 

Include the reasons why you're requesting the new feature and how it might benefit other Crucible users.

## License

Copyright 2021 Carnegie Mellon University. See the [LICENSE.md](./LICENSE.md) files for details.
