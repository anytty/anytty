package state

type HostThemeStore struct {
	DefaultFG string
	DefaultBG string
	Palette   map[int]string
	Probed    bool
}

type HostThemeUpdate struct {
	DefaultFG    string
	DefaultBG    string
	PaletteIndex int
	PaletteColor string
}

func (store HostThemeStore) ApplyUpdate(update HostThemeUpdate) HostThemeStore {
	if update.DefaultFG != "" {
		store.DefaultFG = update.DefaultFG
		store.Probed = true
	}
	if update.DefaultBG != "" {
		store.DefaultBG = update.DefaultBG
		store.Probed = true
	}
	if update.PaletteColor != "" {
		if store.Palette == nil {
			store.Palette = map[int]string{}
		} else {
			cloned := make(map[int]string, len(store.Palette)+1)
			for index, color := range store.Palette {
				cloned[index] = color
			}
			store.Palette = cloned
		}
		store.Palette[update.PaletteIndex] = update.PaletteColor
		store.Probed = true
	}
	return store
}

func (store HostThemeStore) PaletteColor(index int) (string, bool) {
	if store.Palette == nil {
		return "", false
	}
	color, ok := store.Palette[index]
	return color, ok
}
