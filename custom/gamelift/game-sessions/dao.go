package gamesessions

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/gamelift"
	"github.com/aws/aws-sdk-go-v2/service/gamelift/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

// GameSessionDAO provides data access for GameLift game sessions.
type GameSessionDAO struct {
	dao.BaseDAO
	client *gamelift.Client
}

// NewGameSessionDAO creates a new GameSessionDAO.
func NewGameSessionDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}
	return &GameSessionDAO{
		BaseDAO: dao.NewBaseDAO("gamelift", "game-sessions"),
		client:  gamelift.NewFromConfig(cfg),
	}, nil
}

// List returns GameLift game sessions, filtered by FleetId from context.
func (d *GameSessionDAO) List(ctx context.Context) ([]dao.Resource, error) {
	fleetId := dao.GetFilterFromContext(ctx, "FleetId")
	if fleetId == "" {
		return nil, fmt.Errorf("FleetId filter is required to list game sessions; navigate from a fleet")
	}

	sessions, err := appaws.Paginate(ctx, func(token *string) ([]types.GameSession, *string, error) {
		output, err := d.client.DescribeGameSessions(ctx, &gamelift.DescribeGameSessionsInput{
			FleetId:   &fleetId,
			NextToken: token,
		})
		if err != nil {
			return nil, nil, apperrors.Wrap(err, "describe gamelift game sessions")
		}
		return output.GameSessions, output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	resources := make([]dao.Resource, len(sessions))
	for i, session := range sessions {
		resources[i] = NewGameSessionResource(session)
	}
	return resources, nil
}

// Get returns a specific GameLift game session by ID.
func (d *GameSessionDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	output, err := d.client.DescribeGameSessions(ctx, &gamelift.DescribeGameSessionsInput{
		GameSessionId: &id,
	})
	if err != nil {
		return nil, apperrors.Wrapf(err, "describe gamelift game session %s", id)
	}
	if len(output.GameSessions) == 0 {
		return nil, fmt.Errorf("gamelift game session %s not found", id)
	}
	return NewGameSessionResource(output.GameSessions[0]), nil
}

// Delete is not supported for game sessions.
func (d *GameSessionDAO) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("delete is not supported for game sessions")
}

// Supports returns whether this DAO supports the given operation.
func (d *GameSessionDAO) Supports(op dao.Operation) bool {
	switch op {
	case dao.OpList, dao.OpGet:
		return true
	default:
		return false
	}
}

// GameSessionResource wraps a GameLift game session.
type GameSessionResource struct {
	dao.BaseResource
	Session types.GameSession
}

// NewGameSessionResource creates a new GameSessionResource.
func NewGameSessionResource(session types.GameSession) *GameSessionResource {
	sessionId := appaws.Str(session.GameSessionId)
	name := appaws.Str(session.Name)
	if name == "" {
		name = appaws.ExtractResourceName(sessionId)
	}
	return &GameSessionResource{
		BaseResource: dao.BaseResource{
			ID:   sessionId,
			Name: name,
			ARN:  sessionId, // GameSessionId is the ARN
			Data: session,
		},
		Session: session,
	}
}

// Status returns the game session status.
func (r *GameSessionResource) Status() string {
	return string(r.Session.Status)
}

// StatusReason returns the status reason.
func (r *GameSessionResource) StatusReason() string {
	return string(r.Session.StatusReason)
}

// FleetId returns the fleet ID.
func (r *GameSessionResource) FleetId() string {
	return appaws.Str(r.Session.FleetId)
}

// FleetArn returns the fleet ARN.
func (r *GameSessionResource) FleetArn() string {
	return appaws.Str(r.Session.FleetArn)
}

// IpAddress returns the IP address.
func (r *GameSessionResource) IpAddress() string {
	return appaws.Str(r.Session.IpAddress)
}

// DnsName returns the DNS name.
func (r *GameSessionResource) DnsName() string {
	return appaws.Str(r.Session.DnsName)
}

// Port returns the port number.
func (r *GameSessionResource) Port() int32 {
	return appaws.Int32(r.Session.Port)
}

// CurrentPlayerSessionCount returns the current player count.
func (r *GameSessionResource) CurrentPlayerSessionCount() int32 {
	return appaws.Int32(r.Session.CurrentPlayerSessionCount)
}

// MaximumPlayerSessionCount returns the max player count.
func (r *GameSessionResource) MaximumPlayerSessionCount() int32 {
	return appaws.Int32(r.Session.MaximumPlayerSessionCount)
}

// PlayerSessionCreationPolicy returns the player session creation policy.
func (r *GameSessionResource) PlayerSessionCreationPolicy() string {
	return string(r.Session.PlayerSessionCreationPolicy)
}

// CreatorId returns the creator ID.
func (r *GameSessionResource) CreatorId() string {
	return appaws.Str(r.Session.CreatorId)
}

// Location returns the fleet location.
func (r *GameSessionResource) Location() string {
	return appaws.Str(r.Session.Location)
}

// CreationTime returns when the session was created.
func (r *GameSessionResource) CreationTime() *time.Time {
	return r.Session.CreationTime
}

// TerminationTime returns when the session was terminated.
func (r *GameSessionResource) TerminationTime() *time.Time {
	return r.Session.TerminationTime
}

// GameProperties returns the game properties.
func (r *GameSessionResource) GameProperties() []types.GameProperty {
	return r.Session.GameProperties
}

// GameSessionData returns the game session data.
func (r *GameSessionResource) GameSessionData() string {
	return appaws.Str(r.Session.GameSessionData)
}
