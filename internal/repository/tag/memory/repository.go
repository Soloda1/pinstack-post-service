package memory

import (
	"context"
	"sync"

	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"
)

type TagRepository struct {
	log          *logger.Logger
	mu           sync.RWMutex
	tags         map[int64]*model.Tag
	tagsByName   map[string]*model.Tag
	postTags     map[int64]map[int64]bool
	postsByTagID map[int64]map[int64]bool
	postExists   map[int64]bool
	nextID       int64
}

func NewTagRepository(log *logger.Logger) *TagRepository {
	return &TagRepository{
		log:          log,
		tags:         make(map[int64]*model.Tag),
		tagsByName:   make(map[string]*model.Tag),
		postTags:     make(map[int64]map[int64]bool),
		postsByTagID: make(map[int64]map[int64]bool),
		postExists:   make(map[int64]bool),
		nextID:       1,
	}
}

func (t *TagRepository) SimulatePostExists(postID int64, exists bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.postExists[postID] = exists
}

func (t *TagRepository) FindByNames(ctx context.Context, names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*model.Tag
	for _, name := range names {
		if tag, exists := t.tagsByName[name]; exists {
			tagCopy := *tag
			result = append(result, &tagCopy)
		}
	}

	return result, nil
}

func (t *TagRepository) FindByPost(ctx context.Context, postID int64) ([]*model.Tag, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*model.Tag
	if tagMap, exists := t.postTags[postID]; exists {
		for tagID := range tagMap {
			if tag, found := t.tags[tagID]; found {
				tagCopy := *tag
				result = append(result, &tagCopy)
			}
		}
	}

	return result, nil
}

func (t *TagRepository) Create(ctx context.Context, name string) (*model.Tag, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if tag, exists := t.tagsByName[name]; exists {
		tagCopy := *tag
		return &tagCopy, nil
	}

	tag := &model.Tag{
		ID:   t.nextID,
		Name: name,
	}
	t.nextID++

	t.tags[tag.ID] = tag
	t.tagsByName[tag.Name] = tag
	t.postsByTagID[tag.ID] = make(map[int64]bool)

	tagCopy := *tag
	return &tagCopy, nil
}

func (t *TagRepository) DeleteUnused(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for tagID, postMap := range t.postsByTagID {
		if len(postMap) == 0 {
			if tag, exists := t.tags[tagID]; exists {
				delete(t.tagsByName, tag.Name)
				delete(t.tags, tagID)
				delete(t.postsByTagID, tagID)
			}
		}
	}

	return nil
}

func (t *TagRepository) TagPost(ctx context.Context, postID int64, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if exists, found := t.postExists[postID]; !found || !exists {
		return custom_errors.ErrPostNotFound
	}

	if _, exists := t.postTags[postID]; !exists {
		t.postTags[postID] = make(map[int64]bool)
	}

	for _, tagName := range tagNames {
		var tag *model.Tag
		if existingTag, exists := t.tagsByName[tagName]; exists {
			tag = existingTag
		} else {
			tag = &model.Tag{
				ID:   t.nextID,
				Name: tagName,
			}
			t.nextID++
			t.tags[tag.ID] = tag
			t.tagsByName[tagName] = tag
			t.postsByTagID[tag.ID] = make(map[int64]bool)
		}

		t.postTags[postID][tag.ID] = true
		t.postsByTagID[tag.ID][postID] = true
	}

	return nil
}

func (t *TagRepository) UntagPost(ctx context.Context, postID int64, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if exists, found := t.postExists[postID]; !found || !exists {
		return custom_errors.ErrPostNotFound
	}

	for _, tagName := range tagNames {
		if tag, exists := t.tagsByName[tagName]; exists {
			if postTags, found := t.postTags[postID]; found {
				delete(postTags, tag.ID)
			}
			if tagPosts, found := t.postsByTagID[tag.ID]; found {
				delete(tagPosts, postID)
			}
		}
	}

	return nil
}

func (t *TagRepository) ReplacePostTags(ctx context.Context, postID int64, newTags []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if exists, found := t.postExists[postID]; !found || !exists {
		return custom_errors.ErrPostNotFound
	}

	if oldTags, exists := t.postTags[postID]; exists {
		for tagID := range oldTags {
			if tagPosts, found := t.postsByTagID[tagID]; found {
				delete(tagPosts, postID)
			}
		}
	}

	t.postTags[postID] = make(map[int64]bool)

	for _, tagName := range newTags {
		var tag *model.Tag
		if existingTag, exists := t.tagsByName[tagName]; exists {
			tag = existingTag
		} else {
			tag = &model.Tag{
				ID:   t.nextID,
				Name: tagName,
			}
			t.nextID++
			t.tags[tag.ID] = tag
			t.tagsByName[tagName] = tag
			t.postsByTagID[tag.ID] = make(map[int64]bool)
		}

		t.postTags[postID][tag.ID] = true
		if _, exists := t.postsByTagID[tag.ID]; !exists {
			t.postsByTagID[tag.ID] = make(map[int64]bool)
		}
		t.postsByTagID[tag.ID][postID] = true
	}

	return nil
}
