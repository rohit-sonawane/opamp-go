package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/open-telemetry/opamp-go/internal/examples/server/data"
	"github.com/open-telemetry/opamp-go/internal/examples/server/opampsrv"
	"github.com/open-telemetry/opamp-go/internal/examples/server/uisrv"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"
)

var logger = log.New(log.Default().Writer(), "[MAIN] ", log.Default().Flags()|log.Lmsgprefix|log.Lmicroseconds)

var OpampSrv *opampsrv.Server

func startMockServer() {
	curDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	uisrv.Start(curDir)
	OpampSrv = opampsrv.NewServer(&data.AllAgents)
	OpampSrv.Start()
}

func stopMockServer() {
	log.Println("OpAMP Server shutting down...")
	uisrv.Shutdown()
	if OpampSrv != nil {
		OpampSrv.Stop()
	}
}

func sendServerToAgentMessage() {
	// Create a new ServerToAgent message
	msg := &protobufs.ServerToAgent{
		ErrorResponse: &protobufs.ServerErrorResponse{
			ErrorMessage: "Hello there hard coded !!!",
		},
	}

	// Marshal the message to a binary format
	data, err := proto.Marshal(msg)
	if err != nil {

		logger.Printf("Failed to marshal message: %v\n", err)
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "http://localhost:8080/nextresponse", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	// Set the content type to application/x-protobuf
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		return
	}

	fmt.Println("Message sent successfully")
}

func Run() {

	// Initialize Gin router
	r := gin.Default()

	// Define a simple health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// Update the received servertoagent message and update the nextresponse attr
	r.POST("/nextresponse", func(c *gin.Context) {
		// body, err := ioutil.ReadAll(c.Request.Body)

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		var serverToAgent protobufs.ServerToAgent

		if err := proto.Unmarshal(body, &serverToAgent); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request body"})
			return
		}

		// will get object of response
		OpampSrv.UpdateNextResponse(&serverToAgent)
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	// send all agents info
	r.GET("/agents", func(c *gin.Context) {
		sendServerToAgentMessage()
		response := OpampSrv.GetAllAgents()
		json_response, _ := json.MarshalIndent(response, "", "  ")

		log.Printf("response::  %v ", response)
		//send all agents info
		c.JSON(http.StatusOK, gin.H{"response": json_response})
	})

	// ingest test value or nextresponse
	r.GET("/ingest", func(c *gin.Context) {
		sendServerToAgentMessage()
		json_response, _ := json.MarshalIndent("OK", "", "  ")

		c.JSON(http.StatusOK, gin.H{"response": json_response})
	})

	// Define an endpoint to interact with the OpAMP server
	r.POST("/opamp", func(c *gin.Context) {
		// This is where you would handle OpAMP protocol requests
		response := OpampSrv.GetNextResponse()
		if response != nil {
			c.JSON(http.StatusOK, gin.H{"message": response.ErrorResponse.ErrorMessage})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Nothing was found "})
		}

	})

	// Define a shutdown endpoint
	r.POST("/shutdown", func(c *gin.Context) {
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
		c.JSON(http.StatusOK, gin.H{"status": "Shutting down"})
	})
	// Start the mock server
	startMockServer()
	defer stopMockServer()

	// Run the Gin server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run Gin server: %v", err)
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	stopMockServer()
}

func main() {
	Run()
}
