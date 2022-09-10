package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	blogpb "github.com/This-Is-Prince/learning-grpc/mongodb-crud-with-grpc/pb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/joho/godotenv"
)

// Global variables for db connection , collection and context
var db *mongo.Client
var blogdb *mongo.Collection
var mongoCtx context.Context

type BlogServiceServer struct {
	blogpb.UnimplementedBlogServiceServer
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

type BlogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func (s *BlogServiceServer) CreateBlog(ctx context.Context, req *blogpb.CreateBlogReq) (*blogpb.CreateBlogRes, error) {
	// Essentially doing req.Blog to access the struct with a nil check
	blog := req.GetBlog()

	// Now we have to convert this into a BlogItem type to convert into BSON
	data := BlogItem{
		// ID: Empty, so it gets omitted and MongoDB generates a unique Object ID upon insertion.
		AuthorID: blog.GetAuthorId(),
		Content:  blog.GetContent(),
		Title:    blog.GetTitle(),
	}

	// Insert the data into the database, result contains the newly generated Object ID for the new document
	result, err := blogdb.InsertOne(mongoCtx, data)
	// check for potential errors
	if err != nil {
		// return internal gRPC error to be handled later
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Internal error: %v", err))
	}

	// add the id to blog, first cast the "generic type" (go doesn't have real generics yet) to an Object ID.
	oid := result.InsertedID.(primitive.ObjectID)

	// Convert the object id to it's string counterpart
	blog.Id = oid.Hex()

	// return the blog in a CreateBlogRes type
	return &blogpb.CreateBlogRes{Blog: blog}, nil
}

func (s *BlogServiceServer) ReadBlog(ctx context.Context, req *blogpb.ReadBlogReq) (*blogpb.ReadBlogRes, error) {
	// convert string id (from proto) to mongoDB ObjectId
	oid, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}
	result := blogdb.FindOne(ctx, bson.M{"_id": oid})
	// Create an empty BlogItem to write our decode result to
	data := BlogItem{}
	// decode and write to data
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find blog with Object Id %s: %v", req.GetId(), err))
	}
	// Cast to ReadBlogRes type
	response := &blogpb.ReadBlogRes{
		Blog: &blogpb.Blog{
			Id:       req.GetId(),
			AuthorId: data.AuthorID,
			Title:    data.Title,
			Content:  data.Content,
		},
	}
	return response, nil
}
func (s *BlogServiceServer) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogReq) (*blogpb.UpdateBlogRes, error) {
	// Get the blog data from the request
	blog := req.GetBlog()

	// Convert the Id string to a MongoDB ObjectId
	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied blog id to a MongoDB ObjectId: %v", err))
	}

	// Convert the data to be updated into an unordered BSON document
	update := bson.M{
		"author_id": blog.GetAuthorId(),
		"title":     blog.GetTitle(),
		"content":   blog.GetContent(),
	}

	// Convert the oid into an unordered BSON document to search by id
	filter := bson.M{
		"_id": oid,
	}

	// Result is the BSON encoded result
	// To return the updated document instead of original we have to add options.
	result := blogdb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))

	// Decode result and write it to `decoded`
	decoded := BlogItem{}
	err = result.Decode(&decoded)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find blog with supplied ID: %v", err))
	}

	return &blogpb.UpdateBlogRes{
		Blog: &blogpb.Blog{
			Id:       decoded.ID.Hex(),
			AuthorId: decoded.AuthorID,
			Title:    decoded.Title,
			Content:  decoded.Content,
		},
	}, nil
}
func (s *BlogServiceServer) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogReq) (*blogpb.DeleteBlogRes, error) {
	// Get the ID (string) from the request message and convert it to an Object ID
	oid, err := primitive.ObjectIDFromHex(req.GetId())
	// Check for errors
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}
	// DeleteOne returns DeleteResult which is a struct containing the amount of deleted docs (in this case only 1 always)
	// So we return a boolean instead
	_, err = blogdb.DeleteOne(ctx, bson.M{"_id": oid})
	// Check for errors
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find/delete blog with id %s: %v", req.GetId(), err))
	}
	// Return response with success: true if no error is thrown (and thus document is removed)
	return &blogpb.DeleteBlogRes{
		Success: true,
	}, nil

}
func (s *BlogServiceServer) ListBlogs(req *blogpb.ListBlogReq, stream blogpb.BlogService_ListBlogsServer) error {
	// Initiate a BlogItem type to write decoded data to
	data := &BlogItem{}
	// collection.Find returns a cursor for our (empty) query
	cursor, err := blogdb.Find(context.Background(), bson.M{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	// An expression with defer will be called at the end of the function
	defer cursor.Close(context.Background())
	// cursor.Next() returns a boolean, if false there are no more items and loop will break
	for cursor.Next(context.Background()) {
		// Decode the data at the current pointer and write it to data
		err := cursor.Decode(data)
		// check error
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}
		// If no error is found send blog over stream
		stream.Send(&blogpb.ListBlogRes{
			Blog: &blogpb.Blog{
				Id:       data.ID.Hex(),
				AuthorId: data.AuthorID,
				Content:  data.Content,
				Title:    data.Title,
			},
		})
	}
	// Check if the cursor has any errors
	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown cursor error: %v", err))
	}
	return nil
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
	c := make(chan os.Signal, 1)

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
