// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

// Interact with the persistent mapping from ContentID,
// ContentID to sha of the content tree

package volumemgr

import (
	"github.com/lf-edge/eve/pkg/pillar/types"
	"github.com/lf-edge/eve/pkg/pillar/utils"
	uuid "github.com/satori/go.uuid"
)

// Add or update
func latchContentTreeHash(ctx *volumemgrContext, contentID uuid.UUID,
	hash string, generationCounter uint32) {

	log.Functionf("latchContentTreeHash(%s, %s, %d)", contentID, hash, generationCounter)
	if hash == "" {
		log.Errorf("latchContentTreeHash(%s, %d) empty hash",
			contentID, generationCounter)
		return
	}
	aih := types.AppAndImageToHash{
		ImageID:      contentID,
		Hash:         hash,
		PurgeCounter: generationCounter,
	}
	item, _ := ctx.pubContentTreeToHash.Get(aih.Key())
	if item != nil {
		old := item.(types.AppAndImageToHash)
		if old.Hash == aih.Hash {
			log.Warnf("latchContentTreeHash(%s, %d) no change %s",
				contentID, generationCounter, old.Hash)
			return
		}
		log.Warnf("latchContentTreeHash(%s, %d) change from %s to %s",
			contentID, generationCounter, old.Hash, aih.Hash)
	}
	ctx.pubContentTreeToHash.Publish(aih.Key(), aih)
	log.Functionf("latchContentTreeHash(%s, %s, %d) done", contentID, hash, generationCounter)
}

// Delete for a specific content tree
func deleteLatchContentTreeHash(ctx *volumemgrContext,
	contentID uuid.UUID, generationCounter uint32) {

	log.Functionf("deleteLatchContentTreeHash(%s, %d)", contentID, generationCounter)
	aih := types.AppAndImageToHash{
		ImageID:      contentID,
		PurgeCounter: generationCounter,
	}
	item, _ := ctx.pubContentTreeToHash.Get(aih.Key())
	if item == nil {
		log.Errorf("deleteLatchContentTreeHash(%s, %d) not found",
			contentID, generationCounter)
		return
	}
	ctx.pubContentTreeToHash.Unpublish(aih.Key())
	log.Functionf("deleteLatchContentTreeHash(%s, %d) done", contentID, generationCounter)
}

// Purge all for contentID
func purgeLatchContentTreeHash(ctx *volumemgrContext, contentID uuid.UUID) {

	log.Functionf("purgeLatchContentTreeHash(%s)", contentID)
	items := ctx.pubContentTreeToHash.GetAll()
	for _, a := range items {
		aih := a.(types.AppAndImageToHash)
		if aih.ImageID == contentID {
			log.Errorf("purgeLatchContentTreeHash(%s) deleting %s hash %s",
				contentID, aih.ImageID, aih.Hash)
			ctx.pubContentTreeToHash.Unpublish(aih.Key())
		}
	}
	log.Functionf("purgeLatchContentTreeHash(%s) done", contentID)
}

// Returns "" string if not found
func lookupLatchContentTreeHash(ctx *volumemgrContext,
	contentID uuid.UUID, generationCounter uint32) string {

	log.Tracef("lookupLatchContentTreeHash(%s, %d)", contentID, generationCounter)
	temp := types.AppAndImageToHash{
		ImageID:      contentID,
		PurgeCounter: generationCounter,
	}
	item, _ := ctx.pubContentTreeToHash.Get(temp.Key())
	if item == nil {
		log.Tracef("lookupLatchContentTreeHash(%s, %d) not found",
			contentID, generationCounter)
		return ""
	}
	aih := item.(types.AppAndImageToHash)
	log.Tracef("lookupLatchContentTreeHash(%s, %d) found %s",
		contentID, generationCounter, aih.Hash)
	return aih.Hash
}

// Can update status
func maybeLatchContentTreeHash(ctx *volumemgrContext, status *types.ContentTreeStatus) {

	imageSha := lookupLatchContentTreeHash(ctx, status.ContentID, uint32(status.GenerationCounter))
	if imageSha == "" {
		if status.IsContainer() && status.ContentSha256 == "" {
			log.Functionf("ContentTree(%s) %s has not (yet) latched sha",
				status.ContentID, status.DisplayName)
		}
		return
	}
	if status.ContentSha256 == "" {
		log.Functionf("Latching ContentTree(%s) %s to sha %s",
			status.ContentID, status.DisplayName, imageSha)
		status.ContentSha256 = imageSha
		if status.IsContainer() {
			newName := utils.MaybeInsertSha(status.RelativeURL, imageSha)
			if newName != status.RelativeURL {
				log.Functionf("Changing content tree name from %s to %s",
					status.RelativeURL, newName)
				status.RelativeURL = newName
			}
		}
	} else if status.ContentSha256 != imageSha {
		// We already catch this change, but logging here in any case
		log.Warnf("ContentTree(%s) %s hash sha %s received %s",
			status.ContentID, status.DisplayName,
			imageSha, status.ContentSha256)
	}
}
