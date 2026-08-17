package git

import (
	"encoding/json"
	"errors"
	"strings"
)

type GitLabPushEvent struct {
	ObjectKind string `json:"object_kind"`
	Ref        string `json:"ref"`
}

func ParseGitLabPushEvent(payload []byte) (*GitLabPushEvent, error) {
	var ev GitLabPushEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if ev.ObjectKind != "push" {
		return nil, errors.New("not a push event")
	}
	return &ev, nil
}

func (e *GitLabPushEvent) Branch() string {
	return strings.TrimPrefix(e.Ref, "refs/heads/")
}

type GitLabMergeEvent struct {
	ObjectKind       string `json:"object_kind"`
	ObjectAttributes struct {
		Action       string `json:"action"`
		SourceBranch string `json:"source_branch"`
		State        string `json:"state"`
	} `json:"object_attributes"`
}

func ParseGitLabMergeEvent(payload []byte) (*GitLabMergeEvent, error) {
	var ev GitLabMergeEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if ev.ObjectKind != "merge_request" {
		return nil, errors.New("not a merge request")
	}
	return &ev, nil
}

type BitbucketPushEvent struct {
	Push struct {
		Changes []struct {
			New struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"new"`
		} `json:"changes"`
	} `json:"push"`
}

func ParseBitbucketPushEvent(payload []byte) (*BitbucketPushEvent, error) {
	var ev BitbucketPushEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if len(ev.Push.Changes) == 0 {
		return nil, errors.New("no changes")
	}
	return &ev, nil
}

func (e *BitbucketPushEvent) Branch() string {
	for _, ch := range e.Push.Changes {
		if ch.New.Type == "branch" && ch.New.Name != "" {
			return ch.New.Name
		}
	}
	return ""
}

type BitbucketPREvent struct {
	PullRequest struct {
		ID     int    `json:"id"`
		State  string `json:"state"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
	} `json:"pullrequest"`
}

func ParseBitbucketPREvent(payload []byte) (*BitbucketPREvent, error) {
	var ev BitbucketPREvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if ev.PullRequest.State == "" {
		return nil, errors.New("not a pull request")
	}
	return &ev, nil
}
