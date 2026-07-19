package tools

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type closeBridgeFixture struct {
	snapshot ChromeSnapshot
	closed   []int
	err      error
}

func (bridge *closeBridgeFixture) Snapshot(context.Context) (ChromeSnapshot, error) {
	return bridge.snapshot, nil
}

func (bridge *closeBridgeFixture) CloseTabs(_ context.Context, tabIDs []int) (TabsCloseResult, error) {
	bridge.closed = append([]int(nil), tabIDs...)
	if bridge.err != nil {
		return TabsCloseResult{}, bridge.err
	}
	return TabsCloseResult{ClosedTabIDs: append([]int(nil), tabIDs...)}, nil
}

func TestCloseTabsRequiresConfirmationBeforeReadingChrome(t *testing.T) {
	bridge := &closeBridgeFixture{}
	_, err := CloseTabs(context.Background(), bridge, TabsCloseInput{TabIDs: []int{7}})
	if !hasErrorCode(err, CodeConfirmationRequired) || bridge.closed != nil {
		t.Fatalf("error=%v closed=%v", err, bridge.closed)
	}
}

func TestCloseTabsValidatesAllTargetsBeforeClosing(t *testing.T) {
	bridge := &closeBridgeFixture{snapshot: ChromeSnapshot{Tabs: []ChromeTab{{ID: 7}}}}
	_, err := CloseTabs(context.Background(), bridge, TabsCloseInput{TabIDs: []int{7, 9}, Confirmed: true})
	if !hasErrorCode(err, CodeTabNotFound) || bridge.closed != nil {
		t.Fatalf("error=%v closed=%v", err, bridge.closed)
	}
	var toolError *Error
	if !errors.As(err, &toolError) || !reflect.DeepEqual(toolError.Details["tabIds"], []int{9}) {
		t.Fatalf("details=%#v", toolError)
	}
}

func TestCloseTabsClosesExactlyTheConfirmedTargets(t *testing.T) {
	bridge := &closeBridgeFixture{snapshot: ChromeSnapshot{Tabs: []ChromeTab{{ID: 7}, {ID: 9}}}}
	result, err := CloseTabs(context.Background(), bridge, TabsCloseInput{TabIDs: []int{9, 7}, Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bridge.closed, []int{9, 7}) || !reflect.DeepEqual(result.ClosedTabIDs, []int{9, 7}) || result.ProtocolVersion != ProtocolVersion {
		t.Fatalf("closed=%v result=%#v", bridge.closed, result)
	}
}

func TestCloseTabsRejectsDuplicateTargets(t *testing.T) {
	bridge := &closeBridgeFixture{}
	_, err := CloseTabs(context.Background(), bridge, TabsCloseInput{TabIDs: []int{7, 7}, Confirmed: true})
	if !hasErrorCode(err, CodeInvalidArgument) || bridge.closed != nil {
		t.Fatalf("error=%v closed=%v", err, bridge.closed)
	}
}

func hasErrorCode(err error, code ErrorCode) bool {
	var toolError *Error
	return errors.As(err, &toolError) && toolError.Code == code
}
