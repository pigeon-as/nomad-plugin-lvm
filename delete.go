package main

func cmdDelete(cfg *Config) error {
	volumeID, err := envRequired("DHV_VOLUME_ID")
	if err != nil {
		return err
	}
	return lvRemove(cfg.VolumeGroup, volumeID)
}
