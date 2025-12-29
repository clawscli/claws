# IAM Permissions

claws requires appropriate IAM permissions to access AWS resources. The permissions needed depend on which services you want to browse.

## Minimum Permissions

For basic read-only browsing, claws needs `Describe*`, `List*`, and `Get*` permissions for the services you want to access.

## Inline Metrics (Optional)

To display inline CloudWatch metrics (toggle with `M` key), you need:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "cloudwatch:GetMetricData",
      "Resource": "*"
    }
  ]
}
```

Metrics are disabled by default. When enabled, claws fetches the last hour of metrics for supported resources (EC2, RDS, Lambda).

## Resource Actions

Some resource actions require additional permissions:

| Action | Permission Required |
|--------|---------------------|
| Start/Stop EC2 | `ec2:StartInstances`, `ec2:StopInstances` |
| Delete resources | `<service>:Delete*` |
| SSO Login | `sso:*` (for SSO profiles) |

## Recommended Policy

For full read-only access with metrics:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "rds:Describe*",
        "lambda:List*",
        "lambda:Get*",
        "s3:List*",
        "s3:GetBucket*",
        "cloudwatch:GetMetricData",
        "iam:List*",
        "iam:Get*"
      ],
      "Resource": "*"
    }
  ]
}
```

For full access, use AWS managed policies like `ReadOnlyAccess` or `ViewOnlyAccess`.

## Read-Only Mode

Run claws with `--read-only` or set `CLAWS_READ_ONLY=1` to disable all destructive actions, regardless of IAM permissions.
