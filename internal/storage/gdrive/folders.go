package gdrive

import (
	"context"
	"errors"
	"mime"
	"path"
	"strings"

	"aether/internal/storage"
)

func (p *Provider) findFolder(ctx context.Context, parentID, name string) (*File, error) {
	out, err := p.client.ListFiles(ctx, ListFilesInput{
		ParentID: parentID, Name: name, MimeType: folderMIME, PageSize: 100,
	})
	if err != nil {
		return nil, mapError(err)
	}
	for _, f := range out.Files {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, nil
}

func (p *Provider) findObject(ctx context.Context, parentID, name string) (*File, error) {
	out, err := p.client.ListFiles(ctx, ListFilesInput{
		ParentID: parentID, Name: name, PageSize: 100,
	})
	if err != nil {
		return nil, mapError(err)
	}
	for _, f := range out.Files {
		if f.Name == name && f.MimeType != folderMIME {
			return f, nil
		}
	}
	return nil, nil
}

func (p *Provider) ensureFolderPath(ctx context.Context, segs []string) (string, error) {
	parent := p.rootID
	for _, seg := range segs {
		f, err := p.findFolder(ctx, parent, seg)
		if err != nil {
			return "", err
		}
		if f == nil {
			created, cerr := p.client.CreateFile(ctx, CreateFileInput{
				Name: seg, MimeType: folderMIME, ParentID: parent,
			})
			if cerr != nil {
				if errors.Is(cerr, storage.ErrObjectAlreadyExists) {
					f, err = p.findFolder(ctx, parent, seg)
					if err != nil {
						return "", err
					}
					if f == nil {
						return "", mapError(cerr)
					}
				} else {
					return "", mapError(cerr)
				}
			} else {
				f = created
			}
		}
		parent = f.ID
	}
	return parent, nil
}

// resolveKey walks the folder hierarchy and returns the file at key, or nil.
func (p *Provider) resolveKey(ctx context.Context, key string) (*File, error) {
	segs := strings.Split(key, "/")
	parent := p.rootID
	for i, seg := range segs {
		last := i == len(segs)-1
		var f *File
		var err error
		if last {
			f, err = p.findObject(ctx, parent, seg)
		} else {
			f, err = p.findFolder(ctx, parent, seg)
		}
		if err != nil {
			return nil, err
		}
		if f == nil {
			return nil, nil
		}
		parent = f.ID
		if last {
			return f, nil
		}
	}
	return nil, nil
}

func detectContentType(name string) string {
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
