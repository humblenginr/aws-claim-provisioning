package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	region          = "us-east-1"
	templateName    = "testing_template"
	serialNumber    = "testing_serial" // Change to the device serial number (this should be the unique identifier for the device. We can use MAC address + a time seeded random sequence of characters
	certificateFile = "device_cert.pem"
	privateKeyFile  = "device_key.pem"
	rootCAFile      = "root_ca.pem" // AWS Root certificate file
	AWSIoTEndpoint  = "aj0bkidxn9p53-ats.iot.us-east-1.amazonaws.com"

	// MQTT Topics
	topicCreateCertificate = "$aws/certificates/create/json"
	topicCreateAccepted    = "$aws/certificates/create/json/accepted"
	topicCreateRejected    = "$aws/certificates/create/json/rejected"
	topicRegisterThing     = "$aws/provisioning-templates/testing_template/provision/json"
	topicRegisterAccepted  = "$aws/provisioning-templates/testing_template/provision/json/accepted"
	topicRegisterRejected  = "$aws/provisioning-templates/testing_template/provision/json/rejected"
)

// Device registration response
type RegisterThingResponse struct {
	DeviceConfiguration map[string]interface{} `json:"deviceConfiguration"`
	ThingName           string                 `json:"thingName"`
}

// Certificate creation response
type CreateCertificateResponse struct {
	CertificateID             string            `json:"certificateId"`
	CertificatePem            string            `json:"certificatePem"`
	PrivateKey                string            `json:"privateKey"`
	CertificateOwnershipToken string            `json:"certificateOwnershipToken"`
	ResourceArns              map[string]string `json:"resourceArns"`
}

func createMQTTClient(certFile, keyFile, rootCAFile string) (mqtt.Client, error) {
	// Load certificates
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %v", err)
	}

	// Load root CA
	rootCA, err := ioutil.ReadFile(rootCAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load root CA: %v", err)
	}

	// Create CA certificate pool
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(rootCA)

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:8883", AWSIoTEndpoint))
	opts.SetTLSConfig(tlsConfig)
	opts.SetClientID(fmt.Sprintf("device-%s", serialNumber))
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Second)

	// Create and connect client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect: %v", token.Error())
	}

	return client, nil
}

/*
Ensure that the device_cert.pem, device_key.pem, and root_ca.pem files are present before running this
*/
func main() {
	log.Println("Starting AWS IoT Device Provisioning test using trusted user flow")

	// 1. Create MQTT client with temporary credentials
	log.Println("Creating MQTT client with temporary credentials...")
	mqttClient, err := createMQTTClient(certificateFile, privateKeyFile, rootCAFile)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}
	defer mqttClient.Disconnect(250)

	// 2. Subscribe to certificate creation response topics
	log.Println("Subscribing to certificate creation response topics...")
	certResponseChan := make(chan CreateCertificateResponse, 1)
	certErrorChan := make(chan error, 1)

	mqttClient.Subscribe(topicCreateAccepted, 1, func(client mqtt.Client, msg mqtt.Message) {
		var response CreateCertificateResponse
		if err := json.Unmarshal(msg.Payload(), &response); err != nil {
			certErrorChan <- fmt.Errorf("failed to unmarshal certificate response: %v", err)
			return
		}
		certResponseChan <- response
	})

	mqttClient.Subscribe(topicCreateRejected, 1, func(client mqtt.Client, msg mqtt.Message) {
		certErrorChan <- fmt.Errorf("certificate creation rejected: %s", string(msg.Payload()))
	})

	// 3. Create permanent certificate via MQTT
	log.Println("Creating permanent certificate via MQTT...")
	createCertPayload := map[string]interface{}{
		"certificateSigningRequest": "", // Empty CSR as we're using AWS IoT to generate keys
	}
	payloadBytes, err := json.Marshal(createCertPayload)
	if err != nil {
		log.Fatalf("Failed to marshal create certificate payload: %v", err)
	}

	token := mqttClient.Publish(topicCreateCertificate, 1, false, payloadBytes)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to publish create certificate request: %v", token.Error())
	}

	// 4. Wait for certificate creation response
	select {
	case certResponse := <-certResponseChan:
		log.Println("Successfully created permanent certificate")
		log.Printf("Certificate ID: %s", certResponse.CertificateID)

		// Save permanent certificate and key
		err = os.WriteFile("permanent_cert.pem", []byte(certResponse.CertificatePem), 0644)
		if err != nil {
			log.Fatalf("Failed to write permanent certificate to file: %v", err)
		}

		err = os.WriteFile("permanent_key.pem", []byte(certResponse.PrivateKey), 0600)
		if err != nil {
			log.Fatalf("Failed to write permanent private key to file: %v", err)
		}

		// 5. Subscribe to thing registration response topics
		log.Println("Subscribing to thing registration response topics...")
		registerResponseChan := make(chan RegisterThingResponse, 1)
		registerErrorChan := make(chan error, 1)

		mqttClient.Subscribe(topicRegisterAccepted, 1, func(client mqtt.Client, msg mqtt.Message) {
			var response RegisterThingResponse
			if err := json.Unmarshal(msg.Payload(), &response); err != nil {
				registerErrorChan <- fmt.Errorf("failed to unmarshal register thing response: %v", err)
				return
			}
			registerResponseChan <- response
		})

		mqttClient.Subscribe(topicRegisterRejected, 1, func(client mqtt.Client, msg mqtt.Message) {
			registerErrorChan <- fmt.Errorf("thing registration rejected: %s", string(msg.Payload()))
		})

		// 6. Register thing via MQTT
		log.Println("Registering thing via MQTT...")
		templateParams := map[string]string{
			"SerialNumber": serialNumber,
		}
		registerThingPayload := map[string]interface{}{
			"certificateOwnershipToken": certResponse.CertificateOwnershipToken,
			"parameters":                templateParams,
		}
		payloadBytes, err = json.Marshal(registerThingPayload)
		if err != nil {
			log.Fatalf("Failed to marshal register thing payload: %v", err)
		}

		token = mqttClient.Publish(topicRegisterThing, 1, false, payloadBytes)
		if token.Wait() && token.Error() != nil {
			log.Fatalf("Failed to publish register thing request: %v", token.Error())
		}

		// 12. Wait for thing registration response
		select {
		case registerResponse := <-registerResponseChan:
			log.Printf("Successfully registered thing: %s", registerResponse.ThingName)
			log.Printf("Device configuration: %+v", registerResponse.DeviceConfiguration)
		case err := <-registerErrorChan:
			log.Fatalf("Thing registration failed: %v", err)
		case <-time.After(10 * time.Second):
			log.Fatal("Timeout waiting for thing registration response")
		}

	case err := <-certErrorChan:
		log.Fatalf("Certificate creation failed: %v", err)
	case <-time.After(10 * time.Second):
		log.Fatal("Timeout waiting for certificate creation response")
	}

	log.Println("Device provisioning test complete")
}
