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

	fileNames, err := cmd.Flags().GetString("fileNames")
	if err != nil {
		fmt.Println("error getting fileNames flag", err)
		os.Exit(1)
	}
	if fileNames == "" {
		fmt.Println("fileNames flag is required")
		os.Exit(1)
	}

	fileNamesSplit := strings.Split(strings.Trim(fileNames, " "), " ")

	fmt.Println("Initiating render request with server")
	fmt.Println("fileNames:", fileNames)
	fmt.Println("repoURL:", repoURL)

	var messageClient mqtt.MqttClient

	user := os.Getenv("CLOUDMQTT_MATH_PROOFS_AGENT_USER")
	pw := os.Getenv("CLOUDMQTT_MATH_PROOFS_AGENT_PASSWORD")
	url := os.Getenv("CLOUDMQTT_URL")

	if user == "" || pw == "" || url == "" {
		panic("CLOUDMQTT_MATH_PROOFS_AGENT_USER CLOUDMQTT_MATH_PROOFS_AGENT_PASSWORD CLOUDMQTT_URL env vars must be set")
	}

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, pw, strings.Split(url, "@")[1])

	clientID := fmt.Sprintf("math-visual-proofs-agent-%d", time.Now().Unix())
	messageClient = mqtt.NewMQTTClient(mqttAddr, clientID, func(client mqttC.Client) {
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
		m := mqtt.RenderFeedbackMessage{}
		err := json.Unmarshal([]byte(message), &m)
		if err != nil {
			fmt.Println("error unmarshalling feedback message from server: ", err)
			os.Exit(1)
		}
		if m.RepoURL == repoURL {
			fmt.Println("server responded: ", m.Message)
			serverAcked = true
		}
	})

	messageClient.Subscribe(mqtt.RenderErrTopic, func(message string) {
		m := mqtt.RenderFeedbackMessage{}
		err := json.Unmarshal([]byte(message), &m)
		if err != nil {
			fmt.Println("error unmarshalling feedback message from server: ", err)
			os.Exit(1)
		}
		if m.RepoURL == repoURL {
			fmt.Println("server encountered an error: ", m.Message)
			os.Exit(1)
		}
	})

	messageClient.Subscribe(mqtt.RenderSuccessTopic, func(message string) {
		m := mqtt.RenderFeedbackMessage{}
		err := json.Unmarshal([]byte(message), &m)
		if err != nil {
			fmt.Println("error unmarshalling feedback message from server: ", err)
			os.Exit(1)
		}
		if m.RepoURL == repoURL {
			fmt.Println("server successfully finshed render: ", m.Message)
			os.Exit(0)
		}
	})

	time.Sleep(1 * time.Second)

	m := mqtt.RenderMessage{
		FileNames: fileNamesSplit,
		RepoURL:   repoURL,
	}

	j, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = messageClient.Publish(mqtt.RenderStartTopic, string(j))
	if err != nil {
		fmt.Println("error initiating render:", err)
		os.Exit(1)
	}

	retry := 0
	maxRetries := 5
	for !serverAcked {
		if retry > maxRetries {
			fmt.Println("reached max retries waiting for server to acknowlege render request")
			os.Exit(1)
		}
		fmt.Println("waiting for server to acknowledge render request, retry ", retry)
		retry++
		time.Sleep(1 * time.Second)
	}

	retry = 0
	maxRetries = 12
	for {
		if retry > maxRetries {
			fmt.Println("reached max retries waiting for server to successfully finish render. This does not necessarily mean the render failed but could still be running")
			os.Exit(0)
		}
		fmt.Println("waiting for server to successfully finish render, retry ", retry)
		retry++
		time.Sleep(10 * time.Second)
	}
}
