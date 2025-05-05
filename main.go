package main

import (
	db "calendar-scaler/database"
	"context"
	"fmt"
	"log"
	"net"

	pb "calendar-scaler/externalscaler"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ExternalScaler struct {
	pb.UnimplementedExternalScalerServer
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {
	databasetype := scaledObject.GetScalerMetadata()["type"]
	database, err := db.NewDatabase(databasetype, scaledObject)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	defer database.Close()

	events, err := database.GetEvents()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.IsActiveResponse{
		Result: events != nil,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: "eventTerm",
			TargetSize: 1,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	databasetype := metricRequest.ScaledObjectRef.GetScalerMetadata()["type"]
	database, err := db.NewDatabase(databasetype, metricRequest.ScaledObjectRef)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	defer database.Close()

	events, err := database.GetEvents()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	maxDesiredReplicas := 0
	for _, event := range events {
		if event.DesiredReplicas > maxDesiredReplicas {
			maxDesiredReplicas = event.DesiredReplicas
		}
	}

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  "eventTerm",
			MetricValue: int64(maxDesiredReplicas),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {
	return status.Error(codes.Internal, "The external-push is not implemented.")
}

func main() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
