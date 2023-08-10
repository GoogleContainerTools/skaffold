/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"errors"
	"fmt"
	"reflect"
)

type pipelineUpgrader struct {
	oldConfig reflect.Value
	newConfig reflect.Value
	upgrade   func(o, n interface{}) error
}

func UpgradePipelines(oldConfig, newConfig interface{}, upgrade func(o, n interface{}) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = fmt.Errorf("upgrading pipelines failed: %s", x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()

	upgrader := pipelineUpgrader{
		oldConfig: reflect.Indirect(reflect.ValueOf(oldConfig)),
		newConfig: reflect.Indirect(reflect.ValueOf(newConfig)),
		upgrade:   upgrade,
	}

	if err := upgrader.mainPipeline(); err != nil {
		return err
	}

	return upgrader.profiles()
}

func (u *pipelineUpgrader) mainPipeline() error {
	const fieldMainPipeline = "Pipeline"

	oldPipeline := u.oldConfig.FieldByName(fieldMainPipeline).Addr().Interface()
	newPipeline := u.newConfig.FieldByName(fieldMainPipeline).Addr().Interface()

	err := u.upgrade(oldPipeline, newPipeline)
	if err != nil {
		return fmt.Errorf("upgrading main pipeline: %w", err)
	}

	return nil
}

func (u *pipelineUpgrader) profiles() error {
	const (
		fieldProfilePipeline = "Pipeline"
		fieldProfiles        = "Profiles"
	)

	profilesOld := u.oldConfig.FieldByName(fieldProfiles)
	profilesNew := u.newConfig.FieldByName(fieldProfiles)

	if profilesOld.Len() != profilesNew.Len() {
		return fmt.Errorf("lengths of old and new profiles differ")
	}

	for i := 0; i < profilesOld.Len(); i++ {
		oldPipeline := profilesOld.Index(i).FieldByName(fieldProfilePipeline).Addr().Interface()
		newPipeline := profilesNew.Index(i).FieldByName(fieldProfilePipeline).Addr().Interface()

		if err := u.upgrade(oldPipeline, newPipeline); err != nil {
			return fmt.Errorf("upgrading pipeline of profile %d: %w", i+1, err)
		}
	}

	return nil
}
