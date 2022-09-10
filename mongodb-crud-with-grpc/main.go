package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	blogpb "github.com/This-Is-Prince/learning-grpc/mongodb-crud-with-grpc/pb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"github.com/joho/godotenv"
)

// Global variables for db connection , collection and context
var db *mongo.Client
var blogdb *mongo.Collection
var mongoCtx context.Context

type BlogServiceServer struct{}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	fmt.Println("MongoDB Crud With GRPC")

	// STEP 1
	// Configure 'log' package to give file name and line number on eg. log.Fatal
	// Pipe flags to one another (log.LstdFLags = log.Ldate | log.Ltime)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Starting server on port :50051...")

	// STEP 2
	// Start our listener, 50051 is the default gRPC port
	listener, err := net.Listen("tcp", ":50051")

	// Handle errors if any
	if err != nil {
		log.Fatalf("Unable to listen on port:50051: %v", err)
	}

	// STEP 3
	// Set options, here we can configure things like TLS support
	opts := []grpc.ServerOption{}

	// Create new gRPC server with (blank) options
	s := grpc.NewServer(opts...)

	// Create BlogService type
	srv := &BlogServiceServer{}

	// Register the service with the server
	blogpb.RegisterBlogServiceServer(s, srv)

	// STEP 4
	// Initialize MongoDB client
	fmt.Println("Connecting to MongoDB...")

	// non-nil empty context
	mongoCtx = context.Background()

	// Connect takes in a context and options, the connection URI is the only option we pass for now
	db, err = mongo.Connect(mongoCtx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))

	// Handle potential errors
	if err != nil {
		log.Fatal(err)
	}

	// Check whether the connection was successful by pinging the MongoDB server
	err = db.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		fmt.Println("Connected to Mongodb")
	}

	// Bind our collection to our global variable for use in other methods
	blogdb = db.Database("mydb").Collection("blog")

	// Start the server in a child routine

	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	fmt.Println("Server successfully started on port:50051")

	// Right way to stop the server using a SHUTDOWN HOOk
	// Create a channel to receive OS signals
	c := make(chan os.Signal)

	// Relay os.Interrupt to our channel (os.Interrupt = CTRL+C)
	// Ignore other incoming signals
	signal.Notify(c, os.Interrupt)

	// Block main routine until a signal is received
	// As long as user doesn't press CTRL+C a message is not passed and our main routine keeps running
	<-c

	// After receiving CTRL+C Properly stop the server
	fmt.Println("\nStopping the server...")
	s.Stop()
	listener.Close()
	fmt.Println("Closing MongoDB connection")
	db.Disconnect(mongoCtx)
	fmt.Println("Done.")
}
