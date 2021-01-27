# Providers

## `foodtruck-provider-infra`
The `foodtruck-provider-infra` provider is an example script that can take download a policy
archive created with the `chef export` command and run `chef-client` in local mode
with that policy archive.

### Specification
The `foodtruck-provider-infra` provider specification allows 2 keys:
- `url` (required): The url of the policy archive to download. This file must
  be gzipped tarball (`.tar.gz`). It will be downloaded and unpacked into the
  default temporary directory. The tarball must have an `out` directory, which
  will be cd'd into when running `chef-client`.
- `json_params` (optional): An optional dictionary of attributes that will be
  passed by file to the `chef-client` command via the `-j` switch.

### Examples
```
{
    "url": "https://example.com/policy.tar.gz",
    "json_params": {
        "foo": "bar"
    }
}
```