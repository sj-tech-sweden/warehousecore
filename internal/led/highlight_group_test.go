package led

import (
	"testing"

	"warehousecore/internal/models"
)

func testController() *models.LEDController {
	return &models.LEDController{
		ID:          1,
		DisplayName: "test-controller",
	}
}

func TestHighlightGroupAddBin(t *testing.T) {
	g := newHighlightGroup(testController())
	g.addBin("A", Bin{BinID: "A-01", Pixels: []int{0, 1}, Color: "#FF0000", Pattern: "solid", Intensity: 200})

	if len(g.shelves) != 1 {
		t.Fatalf("expected 1 shelf, got %d", len(g.shelves))
	}
	shelf, ok := g.shelves["A"]
	if !ok {
		t.Fatal("shelf A not found")
	}
	if len(shelf.Bins) != 1 {
		t.Fatalf("expected 1 bin in shelf A, got %d", len(shelf.Bins))
	}
	if shelf.Bins[0].BinID != "A-01" {
		t.Fatalf("expected bin ID A-01, got %q", shelf.Bins[0].BinID)
	}
}

func TestHighlightGroupAddBinMultipleShelf(t *testing.T) {
	g := newHighlightGroup(testController())
	g.addBin("A", Bin{BinID: "A-01"})
	g.addBin("A", Bin{BinID: "A-02"})
	g.addBin("B", Bin{BinID: "B-01"})

	if len(g.shelves) != 2 {
		t.Fatalf("expected 2 shelves, got %d", len(g.shelves))
	}
	if len(g.shelves["A"].Bins) != 2 {
		t.Fatalf("expected 2 bins in shelf A, got %d", len(g.shelves["A"].Bins))
	}
	if len(g.shelves["B"].Bins) != 1 {
		t.Fatalf("expected 1 bin in shelf B, got %d", len(g.shelves["B"].Bins))
	}
}

func TestHighlightGroupBinCount(t *testing.T) {
	g := newHighlightGroup(testController())

	if g.binCount() != 0 {
		t.Fatalf("expected 0 bins initially, got %d", g.binCount())
	}

	g.addBin("A", Bin{BinID: "A-01"})
	g.addBin("A", Bin{BinID: "A-02"})
	g.addBin("B", Bin{BinID: "B-01"})

	if g.binCount() != 3 {
		t.Fatalf("expected 3 bins, got %d", g.binCount())
	}
}

func TestHighlightGroupToCommand(t *testing.T) {
	g := newHighlightGroup(testController())
	g.addBin("A", Bin{BinID: "A-01", Pixels: []int{0}, Color: "#FF0000", Pattern: "solid", Intensity: 200})
	g.addBin("B", Bin{BinID: "B-01", Pixels: []int{5}, Color: "#00FF00", Pattern: "breathe", Intensity: 150})

	cmd := g.toCommand("wh-test")
	if cmd.Op != "highlight" {
		t.Fatalf("expected op=highlight, got %q", cmd.Op)
	}
	if cmd.WarehouseID != "wh-test" {
		t.Fatalf("expected warehouse_id=wh-test, got %q", cmd.WarehouseID)
	}
	if len(cmd.Shelves) != 2 {
		t.Fatalf("expected 2 shelves in command, got %d", len(cmd.Shelves))
	}
}

func TestHighlightGroupToCommandEmptyShelf(t *testing.T) {
	g := newHighlightGroup(testController())
	cmd := g.toCommand("wh-empty")
	if cmd.Op != "highlight" {
		t.Fatalf("expected op=highlight, got %q", cmd.Op)
	}
	if len(cmd.Shelves) != 0 {
		t.Fatalf("expected 0 shelves for empty group, got %d", len(cmd.Shelves))
	}
}

func TestHighlightGroupToCommandBinsCopied(t *testing.T) {
	g := newHighlightGroup(testController())
	g.addBin("A", Bin{BinID: "A-01", Pixels: []int{1, 2, 3}, Color: "#FFFFFF", Pattern: "blink", Intensity: 255})

	cmd := g.toCommand("wh-copy")

	// Mutating the group's internal bins after toCommand should not affect the command's copy.
	// If toCommand only aliases the slice or its elements, these mutations would show up in cmd.
	g.shelves["A"].Bins[0].Color = "#000000"
	g.shelves["A"].Bins = append(g.shelves["A"].Bins, Bin{BinID: "A-02", Pixels: []int{4, 5, 6}, Color: "#123456", Pattern: "solid", Intensity: 100})

	for _, shelf := range cmd.Shelves {
		// We originally added only one bin to shelf "A"; the command should still see exactly one.
		if len(shelf.Bins) != 1 {
			t.Fatalf("command bins share slice header with group bins - expected 1 bin, got %d", len(shelf.Bins))
		}
		for _, b := range shelf.Bins {
			if b.Color == "#000000" {
				t.Fatal("command bins share memory with group bins - expected a copy")
			}
		}
	}
}
