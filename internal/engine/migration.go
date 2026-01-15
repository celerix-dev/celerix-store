package engine

import "fmt"

// Migrate takes data from a source store and pushes it to a destination store.
// This works for:
// - Embedded -> Remote (The "Upgrade")
// - Remote -> Embedded (The "Backup/Offline")
func Migrate(src CelerixStore, dst CelerixStore) error {
	// 1. Get all Personas from the source
	personas, err := src.GetPersonas()
	if err != nil {
		return fmt.Errorf("failed to list personas: %w", err)
	}

	for _, pID := range personas {
		// 2. Get all Apps for this Persona
		apps, err := src.GetApps(pID)
		if err != nil {
			return fmt.Errorf("failed to list apps for persona %s: %w", pID, err)
		}

		for _, aID := range apps {
			// 3. Get the full KV map for this App
			data, err := src.GetAppStore(pID, aID)
			if err != nil {
				return fmt.Errorf("failed to dump data for app %s: %w", aID, err)
			}

			// 4. Push every key into the destination
			for k, v := range data {
				err := dst.Set(pID, aID, k, v)
				if err != nil {
					return fmt.Errorf("failed to set key %s in destination: %w", k, err)
				}
			}
		}
	}

	return nil
}
