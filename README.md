# Jump | Cased Shell prompt auto-detection

Jump is a auto-discovery agent designed to run alongside of Cased Shell. It reads YAML files containing queries to run against compute providers (EC2 and ECS are supported today), executes those queries, and writes a JSON manifest containing the results of those queries. The manifest is updated every 30s as the agent runs. Cased Shell will read this manifest and display the results on the Dashboard.

## Usage

```shell
./jump queries.yaml [queries2.yaml ...] results.json
```

## Writing Queries

Queries have several components:

- `provider`: The provider to query. `ecs`, `ec2`, and `static` are currently supported.
- `filters`: A list of filters to apply to the query. Arguments vary by provider. See the [providers](#providers) section for more information.)
- `limit`, `sortOrder`, and `sortBy`: Optional arguments to limit the results, sort the results, and sort the results by a particular field.
- `prompt`: Metadata to apply to all results returned by this query.
  - `hostname`: The hostname to SSH to when connecting to the prompt. Useful for injecting a jump host into the prompt if necessary.
  - `ipAddress`: The IP address to SSH to when connecting to the prompt. Overrides `hostname`.
  - `port`: The port to use when connecting to the prompt.
  - `username`: The username to use when establishing an SSH connection to the prompt.
  - `name`: A descriptive name for the prompt.
  - `description`: A longer description of the prompt.
  - `jumpCommand`: A command to execute immediately after connecting to the host over SSH. Example: `docker exec -it app-container`
  - `shellCommand`: A command to execute immediately after running `jumpCommand`. Example: `./bin/rails console`
  - `preDownloadCommand`: A command to run before processing a user request to download a file. The tokens `{filepath}` and `{filename}` will be replaced with the full path and filename of the file to download. The command is expected to output the path to the file to download to stdout.
  - `kind`: The "kind" of prompt this is. Currently supported values are "container" and "host".
  - `featured`: Set to true to display this prompt above the fold on the Cased Shell Dashboard.
  - `labels`: A list of key/value pairs describing key characteristics of this prompt. The Cased Shell Dashboard will support filtering prompts by these labels.
  - `annotations`: A list of key/value pairs describing additional characteristics of this prompt. The Cased Shell Dashboard will NOT support filtering prompts by these labels, but may display them for additional context.
  - `principals`: A list of Principals that are known to be allowed to access this prompt. If present, the Cased Shell Dashboard will only display prompts to IDP users that are authorized one of these Principals, for example by membership in a group.
  - `promptForKey`: A boolean that indicates whether or not to prompt for an SSH key when connecting to the prompt.
  - `promptForUsername`: A boolean that indicates whether or not to prompt for a username when connecting to the prompt even if one is set as a default.

### Providers

### `ec2`

#### Filters supported by the `ec2` provider

- `region`: The AWS region to query. Defaults to the current region.

In addition to the above filter keys, the EC2 Provider also accepts all keys that are valid for `ec2.DescribeInstanceInput.Filters`, documentation on which is available at https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#DescribeInstancesInput.

#### Sorting

The EC2 Provider supports sorting by the following keys:

- `launchTime`

### `ecs`

#### Filters supported by the `ecs` provider

- `region`: The AWS region to query. Defaults to the current region.
- `cluster`: The ECS cluster to query. Defaults to the 'default cluster'.
- `task-group`: The name of the ECS Task Group.
- `container-name`: The name of a running Container.

In addition to the above filter keys, the EC2 Provider also accepts all keys that are valid for `ec2.DescribeInstanceInput.Filters`, documentation on which is available at https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#DescribeInstancesInput.

#### Sorting

The ECS Provider supports sorting by the following keys:

- `startedAt`

### `static`

The static provider is a simple provider that does not perform any queries. It is useful for including static prompts along with dynamic ones.

## Example config

```yaml
queries:
  # Include the most recently launched EC2 instance in the `us-west-2` region with an `aws:autoscaling:groupName` tag matching `*bastion*`
  - provider: ec2
    limit: 1
    sortBy: launchTime
    sortOrder: desc
    filters:
      region: us-west-2
      tag:aws:autoscaling:groupName: '*bastion*'
    prompt:
      labels:
        app: bastion
  # Include prompts for all EC2 instances in the `us-west-2` region with an `aws:autoscaling:groupName` tag matching `*prod-cluster*`, configured to proxy their SSH connections through a host in the bastion group above.
  - provider: ec2
    filters:
      region: us-west-2
      tag:aws:autoscaling:groupName: '*prod-cluster*'
    prompt:
      labels:
        cluster: test
      proxyJumpSelector:
        app: bastion
  # Include one featured prompt that allows engineers to connect to a Rails console on the most recently started app container in the production cluster.
  - provider: ecs
    filters:
      region: us-west-2
      cluster: prod-cluster
      container-name: app-container
    limit: 1
    sortBy: startedAt
    sortOrder: desc
    prompt:
      name: Production Rails Console
      description: Use to perform exploratory debugging on the production cluster
      shellCommand: ./bin/rails console
      labels:
        environment: prod
      proxyJumpSelector:
        app: bastion
# Use the static provider to include an additional statically defined prompt.
  - provider: static
    prompt:
      featured: true
      hostname: example.com
      username: example

```
