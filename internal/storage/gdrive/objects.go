package gdrive

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aether/internal/storage"
)

func (p *Provider) PutObject(ctx context.Context, input storage.PutObjectInput) (*storage.PutObjectOutput, error) {
	key, err := p.normalizeKey(input.Key)
	if err != nil {
		return nil, err
	}
	segs := strings.Split(key, "/")
	dir, name := segs[:len(segs)-1], segs[len(segs)-1]

	folderID, err := p.ensureFolderPath(ctx, dir)
	if err != nil {
		return nil, err
	}

	ct := input.ContentType
	if ct == "" {
		ct = detectContentType(name)
	}

	existing, err := p.findObject(ctx, folderID, name)
	if err != nil {
		return nil, err
	}

	var created *File
	if existing != nil {
		created, err = p.client.UpdateFile(ctx, existing.ID, UpdateFileInput{
			MimeType: ct, ContentType: ct, Metadata: input.Metadata, Body: input.Body,
		})
	} else {
		created, err = p.client.CreateFile(ctx, CreateFileInput{
			Name: name, MimeType: ct, ContentType: ct, ParentID: folderID, Metadata: input.Metadata, Body: input.Body,
		})
	}
	if err != nil {
		return nil, mapError(err)
	}
	return &storage.PutObjectOutput{Key: key, Size: created.Size}, nil
}

func (p *Provider) GetObject(ctx context.Context, input storage.GetObjectInput) (*storage.GetObjectOutput, error) {
	key, err := p.normalizeKey(input.Key)
	if err != nil {
		return nil, err
	}
	file, err := p.resolveKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, storage.ErrObjectNotFound
	}
	body, err := p.client.DownloadFile(ctx, file.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return &storage.GetObjectOutput{
		Key:           key,
		Body:          body,
		ContentType:   file.MimeType,
		ContentLength: file.Size,
		LastModified:  file.ModifiedTime,
		Metadata:      file.AppProperties,
	}, nil
}

func (p *Provider) HeadObject(ctx context.Context, input storage.HeadObjectInput) (*storage.HeadObjectOutput, error) {
	key, err := p.normalizeKey(input.Key)
	if err != nil {
		return nil, err
	}
	file, err := p.resolveKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, storage.ErrObjectNotFound
	}
	return &storage.HeadObjectOutput{
		Key:           key,
		ContentLength: file.Size,
		ContentType:   file.MimeType,
		LastModified:  file.ModifiedTime,
		Metadata:      file.AppProperties,
	}, nil
}

func (p *Provider) DeleteObject(ctx context.Context, input storage.DeleteObjectInput) error {
	key, err := p.normalizeKey(input.Key)
	if err != nil {
		return err
	}
	file, err := p.resolveKey(ctx, key)
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	err = p.client.DeleteFile(ctx, file.ID)
	if err != nil {
		if errors.Is(mapError(err), storage.ErrObjectNotFound) {
			return nil
		}
		return mapError(err)
	}
	return nil
}

func (p *Provider) ListObjects(ctx context.Context, input storage.ListObjectsInput) (*storage.ListObjectsOutput, error) {
	state, err := decodeListState(input.Cursor)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state, err = p.initListState(ctx, input.Prefix)
		if err != nil {
			return nil, err
		}
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 1000
	}
	results := make([]storage.ObjectInfo, 0, limit)

	for len(results) < limit && len(state.Folders) > 0 {
		head := state.Folders[0]
		page, perr := p.client.ListFiles(ctx, ListFilesInput{
			ParentID:  head.ID,
			PageToken: state.PageToken,
			PageSize:  int64(min(limit-len(results), 1000)),
		})
		if perr != nil {
			return nil, mapError(perr)
		}
		state.PageToken = ""
		for _, f := range page.Files {
			if head.Filter != "" && !strings.HasPrefix(f.Name, head.Filter) {
				continue
			}
			if f.MimeType == folderMIME {
				state.Folders = append(state.Folders, listFolder{
					ID: f.ID, Path: head.Path + f.Name + "/",
				})
				continue
			}
			results = append(results, storage.ObjectInfo{
				Key:          head.Path + f.Name,
				Size:         f.Size,
				ContentType:  f.MimeType,
				LastModified: f.ModifiedTime,
			})
			if len(results) >= limit {
				break
			}
		}
		if page.NextPageToken != "" {
			state.PageToken = page.NextPageToken
			break
		}
		state.Folders = state.Folders[1:]
	}

	out := &storage.ListObjectsOutput{Objects: results}
	if len(state.Folders) > 0 || state.PageToken != "" {
		out.NextCursor = encodeListState(state)
	}
	return out, nil
}

type listFolder struct {
	ID     string
	Path   string
	Filter string
}

type listState struct {
	Folders   []listFolder
	PageToken string
}

// initListState resolves the prefix to a starting folder and builds the
// initial BFS queue. The last prefix segment acts as a name filter on the
// starting folder's immediate children, mirroring S3 prefix semantics.
func (p *Provider) initListState(ctx context.Context, prefix string) (*listState, error) {
	prefix = strings.TrimPrefix(prefix, "/")
	state := &listState{}

	var dirs []string
	filter := ""
	if prefix != "" {
		segs := strings.Split(prefix, "/")
		if strings.HasSuffix(prefix, "/") {
			dirs = segs[:len(segs)-1]
		} else {
			dirs = segs[:len(segs)-1]
			filter = segs[len(segs)-1]
		}
	}

	folderID := p.rootID
	pathPrefix := ""
	for _, seg := range dirs {
		f, err := p.findFolder(ctx, folderID, seg)
		if err != nil {
			return nil, err
		}
		if f == nil {
			return &listState{}, nil
		}
		folderID = f.ID
		pathPrefix += seg + "/"
	}
	state.Folders = append(state.Folders, listFolder{ID: folderID, Path: pathPrefix, Filter: filter})
	return state, nil
}

func encodeListState(state *listState) string {
	data, err := json.Marshal(state)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeListState(cursor string) (*listState, error) {
	if cursor == "" {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed list cursor", storage.ErrInvalidObjectKey)
	}
	var state listState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("%w: malformed list cursor", storage.ErrInvalidObjectKey)
	}
	return &state, nil
}

func (p *Provider) CopyObject(ctx context.Context, input storage.CopyObjectInput) (*storage.CopyObjectOutput, error) {
	srcKey, err := p.normalizeKey(input.SourceKey)
	if err != nil {
		return nil, err
	}
	dstKey, err := p.normalizeKey(input.DestinationKey)
	if err != nil {
		return nil, err
	}

	src, err := p.resolveKey(ctx, srcKey)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, storage.ErrObjectNotFound
	}

	dstSegs := strings.Split(dstKey, "/")
	dir, name := dstSegs[:len(dstSegs)-1], dstSegs[len(dstSegs)-1]
	folderID, err := p.ensureFolderPath(ctx, dir)
	if err != nil {
		return nil, err
	}

	// S3 copy overwrites the destination; drop a previous file with the key.
	existing, err := p.findObject(ctx, folderID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := p.client.DeleteFile(ctx, existing.ID); err != nil && !errors.Is(mapError(err), storage.ErrObjectNotFound) {
			return nil, mapError(err)
		}
	}

	_, err = p.client.CopyFile(ctx, src.ID, CopyFileInput{Name: name, ParentID: folderID})
	if err != nil {
		return nil, mapError(err)
	}
	return &storage.CopyObjectOutput{
		SourceKey:      srcKey,
		DestinationKey: dstKey,
	}, nil
}
