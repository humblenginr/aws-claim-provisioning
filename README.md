# AWS IoT Device Provisioning Test

This program tests the AWS IoT device provisioning flow using the trusted user method as described in the [AWS IoT documentation](https://docs.aws.amazon.com/iot/latest/developerguide/provision-wo-cert.html#trusted-user).

## Prerequisites

1. AWS CLI configured with appropriate credentials
2. Go 1.20 or later installed
3. An AWS IoT provisioning template named `testing_template` already set up in your AWS account
4. IAM permissions to call the following AWS IoT APIs:
   - `iot:CreateProvisioningClaim`
   - `iot:CreateKeysAndCertificate`
   - `iot:DescribeEndpoint`

## Configuration

Update the following constants in `main.go` before running:

```go
const (
    region           = "us-east-1"     // Change to your AWS region
    templateName     = "testing_template"
    serialNumber     = "device123"     // Change to your device serial number
)
```

## How to Run

```bash
go run main.go
```

## What This Program Does

1. Creates a temporary provisioning claim certificate using your provisioning template
2. Saves the temporary certificate and private key to local files
3. Creates a permanent device certificate
4. Prepares the registration payload that would normally be sent via MQTT
5. Logs the steps that would be performed by a real device implementation

## Files Generated

- `device_cert.pem` - The temporary claim certificate
- `device_key.pem` - The private key for the temporary claim certificate
- `permanent_cert.pem` - The permanent device certificate
- `permanent_key.pem` - The permanent device private key

## In a Real Device Implementation

The program outputs the steps that would be performed on a real device:

1. Connect to AWS IoT using MQTT with the temporary credentials
2. Publish to `$aws/certificates/create/json` to get a permanent certificate
3. Publish to `$aws/provisioning-templates/testing_template/provision/json` with the registration payload
4. Subscribe to `$aws/provisioning-templates/testing_template/provision/json/accepted` to get the response
5. Disconnect and reconnect with the permanent certificate

## Setting Up AWS Resources

Before running this program, you need to set up the following AWS resources:

1. Create a provisioning template in AWS IoT Core (already done as mentioned)
2. Ensure the IAM user/role running this application has the necessary permissions
3. Download the AWS Root CA certificate from [https://www.amazontrust.com/repository/AmazonRootCA1.pem](https://www.amazontrust.com/repository/AmazonRootCA1.pem) and save it as `root_ca.pem` in the same directory

## Example IAM Policy for Running This Program

Here's an example IAM policy that grants the necessary permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "iot:CreateProvisioningClaim",
                "iot:CreateKeysAndCertificate",
                "iot:DescribeEndpoint"
            ],
            "Resource": "*"
        }
    ]
}
```

## Note

This is a test program and simulates the device provisioning flow. In a real-world scenario, this would be implemented on the actual device using MQTT. 