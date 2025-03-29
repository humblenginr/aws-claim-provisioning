# AWS IoT Fleet Provisioning Test - Trusted User Flow

A simple Golang test program demonstrating AWS IoT fleet provisioning using the Trusted User Flow. This program simulates a device using claim certificates to obtain permanent credentials and register with AWS IoT.

## Prerequisites

1. Three certificate files in the project directory:
   - `device_cert.pem` - Initial claim certificate (must be registered with AWS IoT and active)
   - `device_key.pem` - Initial private key for the claim certificate
   - `root_ca.pem` - AWS IoT Root CA ([download here](https://www.amazontrust.com/repository/AmazonRootCA1.pem))
2. An existing AWS IoT provisioning template named `testing_template`
3. Go 1.16 or later installed

## Quick Start

1. Update these constants in `main.go` to match your environment:
   ```go
   const (
       serialNumber    = "testing_serial" // Device identifier to use in provisioning. This can be MAC address + a time based random string
   )
   ```

2. Run the program:
   ```bash
   go run main.go
   ```

## What This Program Does

1. Connects to AWS IoT MQTT using the existing claim certificates
2. Requests a permanent certificate through MQTT
3. Uses the new certificate to register the device with the provisioning template
4. Saves the permanent credentials as `permanent_cert.pem` and `permanent_key.pem`

## Expected Output

When successful, the program will:
1. Create a permanent certificate
2. Register the device with AWS IoT using the template
3. Output the Thing name and device configuration
