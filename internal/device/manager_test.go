package device

import (
	"path/filepath"
	"testing"
)

func TestCustomTargetSurvivesEditAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	d := &Device{Name: "Küchenlicht", Type: TypeSwitch, SwitchMode: SwitchModeOnOff,
		VirtualInputs: map[string]string{"on": "echolox_licht_kueche"}}
	if err := mgr.Create(d); err != nil {
		t.Fatal(err)
	}
	d.Name = "Licht Küche"
	if err := mgr.Update(d); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reloaded.Get(d.ID)
	if !ok || got.VirtualInputs["on"] != "echolox_licht_kueche" {
		t.Fatalf("custom target lost after reload: %+v", got)
	}
}

func TestUpdateWithoutTargetsGeneratesDefaults(t *testing.T) {
	mgr, err := NewManager(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	d := &Device{Name: "Licht", Type: TypeSwitch, SwitchMode: SwitchModeOnOff}
	if err := mgr.Create(d); err != nil {
		t.Fatal(err)
	}
	d.VirtualInputs = nil
	d.Name = "Küche"
	if err := mgr.Update(d); err != nil {
		t.Fatal(err)
	}
	if d.VirtualInputs["on"] != "echolox_kueche" {
		t.Fatalf("unexpected generated targets: %v", d.VirtualInputs)
	}
}
