package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	itypes "github.com/vultisig/verifier/internal/types"
)

func TestAddPluginPolicySync(t *testing.T) {
	t.Skip("Skipping postgres test")
	db, err := NewPostgresBackend("user=myuser password=mypassword dbname=vultisig-verifier host=localhost port=5432 sslmode=disable")
	assert.NoError(t, err)
	ctx := context.Background()
	tx, err := db.Pool().Begin(ctx)
	assert.NoError(t, err)
	syncID := uuid.New()
	err = db.AddPluginPolicySync(ctx, tx, itypes.PluginPolicySync{
		ID:         syncID,
		PolicyID:   uuid.New(),
		PluginID:   uuid.New(),
		SyncType:   itypes.AddPolicy,
		Signature:  "signature",
		Status:     itypes.NotSynced,
		FailReason: "",
	})
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit(ctx))

	syncEntity, err := db.GetPluginPolicySync(ctx, syncID)
	assert.NoError(t, err)
	assert.NotNil(t, syncEntity)
	entities, err := db.GetUnFinishedPluginPolicySyncs(ctx)
	assert.NoError(t, err)
	assert.Len(t, entities, 1)
	syncEntity.Status = itypes.Failed
	tx1, err1 := db.Pool().Begin(ctx)
	assert.NoError(t, err1)
	err = db.UpdatePluginPolicySync(ctx, tx1, *syncEntity)
	assert.NoError(t, err)
	assert.NoError(t, tx1.Commit(ctx))

	assert.NoError(t, db.DeletePluginPolicySync(ctx, syncID))

}
