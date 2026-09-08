package handler

import "github.com/naralabs/naralabs-atlas/internal/module/explore/model"

type NetworkQuery struct {
	Network string `query:"network" doc:"Stellar network (testnet, mainnet, futurenet)" example:"testnet"`
}

type HealthOutput struct {
	Body struct {
		Status string `json:"status" doc:"Service status" example:"ok"`
	}
}

type StatsInput struct {
	NetworkQuery
}

type StatsOutput struct {
	Body model.NetworkStats
}

type EventsInput struct {
	NetworkQuery
	Page         int    `query:"page" minimum:"1" default:"1" doc:"Page number (1-based)"`
	PageSize     int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Number of events per page"`
	Search       string `query:"search" doc:"Search event id, contract id, transaction hash, or payload text"`
	EventType    string `query:"event_type" doc:"Filter by Soroban event type symbol (e.g. transfer, fee)"`
	DecodeStatus string `query:"decode_status" enum:"decoded,raw," doc:"Filter by decode status"`
}

type EventsOutput struct {
	Body model.PaginatedListResponse[model.EventItem]
}

type ContractsInput struct {
	NetworkQuery
	Page         int    `query:"page" minimum:"1" default:"1" doc:"Page number (1-based)"`
	PageSize     int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Number of contracts per page"`
	Search       string `query:"search" doc:"Search by Soroban contract id"`
	SchemaStatus string `query:"schema_status" enum:"decoded,raw_only," doc:"Filter by schema decode status"`
}

type ContractsOutput struct {
	Body model.PaginatedListResponse[model.ContractItem]
}

type HomeInput struct {
	NetworkQuery
	RecentLimit   int `query:"recent_limit" minimum:"1" maximum:"100" default:"8" doc:"Recent events limit for the home payload"`
	ContractLimit int `query:"contract_limit" minimum:"1" maximum:"100" default:"8" doc:"Active contracts limit for the home payload"`
}

type HomeOutput struct {
	Body model.HomePayload
}

type EventDetailInput struct {
	NetworkQuery
	ID string `path:"id" doc:"Soroban event identifier" example:"0001099511627776-0000000001"`
}

type EventDetailOutput struct {
	Body model.EventDetail
}

type ContractDetailInput struct {
	NetworkQuery
	ID string `path:"id" doc:"Soroban contract identifier" example:"CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`
}

type ContractDetailOutput struct {
	Body model.ContractDetail
}

type ContractEventsInput struct {
	NetworkQuery
	ID           string `path:"id" doc:"Soroban contract identifier"`
	Page         int    `query:"page" minimum:"1" default:"1" doc:"Page number (1-based)"`
	PageSize     int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Number of events per page"`
	Search       string `query:"search" doc:"Search event id, transaction hash, or payload text"`
	EventType    string `query:"event_type" doc:"Filter by Soroban event type symbol (e.g. transfer, fee)"`
	DecodeStatus string `query:"decode_status" enum:"decoded,raw," doc:"Filter by decode status"`
}

type ContractEventsOutput struct {
	Body model.PaginatedListResponse[model.EventItem]
}
