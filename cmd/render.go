package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	mqttC "github.com/eclipse/paho.mqtt.golang"

	"github.com/andrewmarklloyd/math-visual-proofs/pkg/mqtt"
	"github.com/spf13/cobra"
)

var serverAcked = false

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "TODO",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		requestRender(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)
}

func requestRender(cmd *cobra.Command, args []string) {
	repoURL, err := cmd.Flags().GetString("repoURL")
	if err != nil {
		fmt.Println("error getting repoURL flag", err)
		os.Exit(1)
	}
	if repoURL == "" {
		fmt.Println("repoURL flag is required")
		os.Exit(1)
	}

	fileName, err := cmd.Flags().GetString("fileName")
	if err != nil {
		fmt.Println("error getting fileName flag", err)
		os.Exit(1)
	}
	if fileName == "" {
		fmt.Println("fileName flag is required")
		os.Exit(1)
	}

	className, err := cmd.Flags().GetString("className")
	if err != nil {
		fmt.Println("error getting fileName flag", err)
		os.Exit(1)
	}
	if className == "" {
		fmt.Println("className flag is required")
		os.Exit(1)
	}

	fmt.Println("Initiating render request with server")
	fmt.Println("className:", className)
	fmt.Println("fileName:", fileName)
	fmt.Println("repoURL:", repoURL)

	var messageClient mqtt.MqttClient

	user := os.Getenv("CLOUDMQTT_MATH_PROOFS_AGENT_USER")
	pw := os.Getenv("CLOUDMQTT_MATH_PROOFS_AGENT_PASSWORD")
	url := os.Getenv("CLOUDMQTT_URL")

	if user == "" || pw == "" || url == "" {
		panic("CLOUDMQTT_MATH_PROOFS_AGENT_USER CLOUDMQTT_MATH_PROOFS_AGENT_PASSWORD CLOUDMQTT_URL env vars must be set")
	}

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, pw, strings.Split(url, "@")[1])

	messageClient = mqtt.NewMQTTClient(mqttAddr, "math-visual-proofs-agent", func(client mqttC.Client) {
		if client.IsConnected() {
			fmt.Println("Connected to MQTT server")
		} else {
			fmt.Println("Not connected to MQTT server")
		}
	}, func(client mqttC.Client, err error) {
		fmt.Println("Connection to MQTT server lost:", err)
	})
	err = messageClient.Connect()
	if err != nil {
		fmt.Println("connecting to mqtt:", err)
	}

	defer messageClient.Cleanup()

	messageClient.Subscribe(mqtt.RenderAckTopic, func(message string) {
		fmt.Println("server responded: ", message)
		serverAcked = true
	})

	time.Sleep(1 * time.Second)

	m := mqtt.RenderMessage{
		FileName:  "MovingAround.py",
		ClassName: "MovingAround",
		RepoURL:   "https://github.com/andrewmarklloyd/math-vis-proofs-test.git",
	}

	j, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = messageClient.Publish(mqtt.RenderStartTopic, string(j))
	if err != nil {
		fmt.Println(err)
	}

	retry := 0
	maxRetries := 10
	for !serverAcked {
		if retry > maxRetries {
			fmt.Println("reached max retries waiting for server to acknowlege render request")
			os.Exit(1)
		}
		fmt.Println("waiting for server to acknowledge render request, retry ", retry)
		retry++
		time.Sleep(5 * time.Second)
	}

}
