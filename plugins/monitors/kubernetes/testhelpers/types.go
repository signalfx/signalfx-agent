package testhelpers

import (
    "k8s.io/client-go/pkg/watch"
    "k8s.io/client-go/pkg/runtime"
)

type WatchEvent struct {
	// The type of the watch event; added, modified, deleted, or error.
	// +optional
	Type watch.EventType `json:"type,omitempty" description:"the type of watch event; may be ADDED, MODIFIED, DELETED, or ERROR"`

	// For added or modified objects, this is the new object; for deleted objects,
	// it's the state of the object immediately prior to its deletion.
	// For errors, it's an api.Status.
	// +optional
	Object runtime.Object `json:"object,omitempty" description:"the object being watched; will match the type of the resource endpoint or be a Status object if the type is ERROR"`
}
