// Copyright 2017 The OpenSDS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
This module implements the etcd database operation of data structure
defined in api module.

*/

package etcd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/golang/glog"
	uuid "github.com/satori/go.uuid"
	c "github.com/sodafoundation/telemetry/pkg/context"
	"github.com/sodafoundation/telemetry/pkg/model"
	"github.com/sodafoundation/telemetry/pkg/utils"
	"github.com/sodafoundation/telemetry/pkg/utils/constants"
	"github.com/sodafoundation/telemetry/pkg/utils/urls"
)

const (
	defaultSortKey          = "ID"
	defaultBlockProfileName = "default_block"
	defaultFileProfileName  = "default_file"
	typeBlock               = "block"
	typeFile                = "file"
)

var validKey = []string{"limit", "offset", "sortDir", "sortKey"}

const (
	typeDocks              string = "Docks"
	typePools              string = "Pools"
	typeProfiles           string = "Profiles"
)

var sortableKeysMap = map[string][]string{
	typeDocks:              {"ID", "NAME", "STATUS", "ENDPOINT", "DRIVERNAME", "DESCRIPTION"},
	typePools:              {"ID", "NAME", "STATUS", "AVAILABILITYZONE", "DOCKID"},
	typeProfiles:           {"ID", "NAME", "DESCRIPTION"},
}

func IsAdminContext(ctx *c.Context) bool {
	return ctx.IsAdmin
}

func AuthorizeProjectContext(ctx *c.Context, tenantId string) bool {
	return ctx.TenantId == tenantId
}

// NewClient
func NewClient(edps []string) *Client {
	return &Client{
		clientInterface: Init(edps),
	}
}

// Client
type Client struct {
	clientInterface
}

//Parameter
type Parameter struct {
	beginIdx, endIdx int
	sortDir, sortKey string
}

//IsInArray
func (c *Client) IsInArray(e string, s []string) bool {
	for _, v := range s {
		if strings.EqualFold(e, v) {
			return true
		}
	}
	return false
}

func (c *Client) SelectOrNot(m map[string][]string) bool {
	for key := range m {
		if !utils.Contained(key, validKey) {
			return true
		}
	}
	return false
}

//Get parameter limit
func (c *Client) GetLimit(m map[string][]string) int {
	var limit int
	var err error
	v, ok := m["limit"]
	if ok {
		limit, err = strconv.Atoi(v[0])
		if err != nil || limit < 0 {
			log.Warning("Invalid input limit:", limit, ",use default value instead:50")
			return constants.DefaultLimit
		}
	} else {
		log.Warning("The parameter limit is not present,use default value instead:50")
		return constants.DefaultLimit
	}
	return limit
}

//Get parameter offset
func (c *Client) GetOffset(m map[string][]string, size int) int {

	var offset int
	var err error
	v, ok := m["offset"]
	if ok {
		offset, err = strconv.Atoi(v[0])

		if err != nil || offset < 0 || offset > size {
			log.Warning("Invalid input offset or input offset is out of bounds:", offset, ",use default value instead:0")

			return constants.DefaultOffset
		}

	} else {
		log.Warning("The parameter offset is not present,use default value instead:0")
		return constants.DefaultOffset
	}
	return offset
}

//Get parameter sortDir
func (c *Client) GetSortDir(m map[string][]string) string {
	var sortDir string
	v, ok := m["sortDir"]
	if ok {
		sortDir = v[0]
		if !strings.EqualFold(sortDir, "desc") && !strings.EqualFold(sortDir, "asc") {
			log.Warning("Invalid input sortDir:", sortDir, ",use default value instead:desc")
			return constants.DefaultSortDir
		}
	} else {
		log.Warning("The parameter sortDir is not present,use default value instead:desc")
		return constants.DefaultSortDir
	}
	return sortDir
}

//Get parameter sortKey
func (c *Client) GetSortKey(m map[string][]string, sortKeys []string) string {
	var sortKey string
	v, ok := m["sortKey"]
	if ok {
		sortKey = strings.ToUpper(v[0])
		if !c.IsInArray(sortKey, sortKeys) {
			log.Warning("Invalid input sortKey:", sortKey, ",use default value instead:ID")
			return defaultSortKey
		}

	} else {
		log.Warning("The parameter sortKey is not present,use default value instead:ID")
		return defaultSortKey
	}
	return sortKey
}

func (c *Client) FilterAndSort(src interface{}, params map[string][]string, sortableKeys []string) interface{} {
	var ret interface{}
	ret = utils.Filter(src, params)
	if len(params["sortKey"]) > 0 && utils.ContainsIgnoreCase(sortableKeys, params["sortKey"][0]) {
		ret = utils.Sort(ret, params["sortKey"][0], c.GetSortDir(params))
	}
	ret = utils.Slice(ret, c.GetOffset(params, reflect.ValueOf(src).Len()), c.GetLimit(params))
	return ret
}

//ParameterFilter
func (c *Client) ParameterFilter(m map[string][]string, size int, sortKeys []string) *Parameter {

	limit := c.GetLimit(m)
	offset := c.GetOffset(m, size)

	beginIdx := offset
	endIdx := limit + offset

	// If use not specified the limit return all the items.
	if limit == constants.DefaultLimit || endIdx > size {
		endIdx = size
	}

	sortDir := c.GetSortDir(m)
	sortKey := c.GetSortKey(m, sortKeys)

	return &Parameter{beginIdx, endIdx, sortDir, sortKey}
}

// CreateDock
func (c *Client) CreateDock(ctx *c.Context, dck *model.DockSpec) (*model.DockSpec, error) {
	if dck.Id == "" {
		dck.Id = uuid.NewV4().String()
	}

	if dck.CreatedAt == "" {
		dck.CreatedAt = time.Now().Format(constants.TimeFormat)
	}

	dckBody, err := json.Marshal(dck)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:     urls.GenerateDockURL(urls.Etcd, "", dck.Id),
		Content: string(dckBody),
	}
	dbRes := c.Create(dbReq)
	if dbRes.Status != "Success" {
		log.Error("when create dock in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	return dck, nil
}

// GetDock
func (c *Client) GetDock(ctx *c.Context, dckID string) (*model.DockSpec, error) {
	dbReq := &Request{
		Url: urls.GenerateDockURL(urls.Etcd, "", dckID),
	}
	dbRes := c.Get(dbReq)
	if dbRes.Status != "Success" {
		log.Error("when get dock in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var dck = &model.DockSpec{}
	if err := json.Unmarshal([]byte(dbRes.Message[0]), dck); err != nil {
		log.Error("when parsing dock in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return dck, nil
}

// GetDockByPoolId
func (c *Client) GetDockByPoolId(ctx *c.Context, poolId string) (*model.DockSpec, error) {
	pool, err := c.GetPool(ctx, poolId)
	if err != nil {
		log.Error("Get pool failed in db: ", err)
		return nil, err
	}

	docks, err := c.ListDocks(ctx)
	if err != nil {
		log.Error("List docks failed failed in db: ", err)
		return nil, err
	}
	for _, dock := range docks {
		if pool.DockId == dock.Id {
			return dock, nil
		}
	}
	return nil, errors.New("Get dock failed by pool id: " + poolId)
}

// ListDocks
func (c *Client) ListDocks(ctx *c.Context) ([]*model.DockSpec, error) {
	dbReq := &Request{
		Url: urls.GenerateDockURL(urls.Etcd, ""),
	}
	dbRes := c.List(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When list docks in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var dcks = []*model.DockSpec{}
	if len(dbRes.Message) == 0 {
		return dcks, nil
	}
	for _, msg := range dbRes.Message {
		var dck = &model.DockSpec{}
		if err := json.Unmarshal([]byte(msg), dck); err != nil {
			log.Error("When parsing dock in db:", dbRes.Error)
			return nil, errors.New(dbRes.Error)
		}
		dcks = append(dcks, dck)
	}
	return dcks, nil
}

func (c *Client) ListDocksWithFilter(ctx *c.Context, m map[string][]string) ([]*model.DockSpec, error) {
	docks, err := c.ListDocks(ctx)
	if err != nil {
		log.Error("List docks failed: ", err.Error())
		return nil, err
	}

	tmpDocks := c.FilterAndSort(docks, m, sortableKeysMap[typeDocks])
	var res = []*model.DockSpec{}
	for _, data := range tmpDocks.([]interface{}) {
		res = append(res, data.(*model.DockSpec))
	}
	return res, nil
}

// UpdateDock
func (c *Client) UpdateDock(ctx *c.Context, dckID, name, desp string) (*model.DockSpec, error) {
	dck, err := c.GetDock(ctx, dckID)
	if err != nil {
		return nil, err
	}
	if name != "" {
		dck.Name = name
	}
	if desp != "" {
		dck.Description = desp
	}
	dck.UpdatedAt = time.Now().Format(constants.TimeFormat)

	dckBody, err := json.Marshal(dck)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:        urls.GenerateDockURL(urls.Etcd, "", dckID),
		NewContent: string(dckBody),
	}
	dbRes := c.Update(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When update dock in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return dck, nil
}

// DeleteDock
func (c *Client) DeleteDock(ctx *c.Context, dckID string) error {
	dbReq := &Request{
		Url: urls.GenerateDockURL(urls.Etcd, "", dckID),
	}
	dbRes := c.Delete(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When delete dock in db:", dbRes.Error)
		return errors.New(dbRes.Error)
	}
	return nil
}

// CreatePool
func (c *Client) CreatePool(ctx *c.Context, pol *model.StoragePoolSpec) (*model.StoragePoolSpec, error) {
	if pol.Id == "" {
		pol.Id = uuid.NewV4().String()
	}

	if pol.CreatedAt == "" {
		pol.CreatedAt = time.Now().Format(constants.TimeFormat)
	}
	polBody, err := json.Marshal(pol)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:     urls.GeneratePoolURL(urls.Etcd, "", pol.Id),
		Content: string(polBody),
	}
	dbRes := c.Create(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When create pol in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	return pol, nil
}

func (c *Client) ListPoolsWithFilter(ctx *c.Context, m map[string][]string) ([]*model.StoragePoolSpec, error) {
	pools, err := c.ListPools(ctx)
	if err != nil {
		log.Error("List pools failed: ", err.Error())
		return nil, err
	}

	tmpPools := c.FilterAndSort(pools, m, sortableKeysMap[typePools])
	var res = []*model.StoragePoolSpec{}
	for _, data := range tmpPools.([]interface{}) {
		res = append(res, data.(*model.StoragePoolSpec))
	}
	return res, nil
}

// GetPool
func (c *Client) GetPool(ctx *c.Context, polID string) (*model.StoragePoolSpec, error) {
	dbReq := &Request{
		Url: urls.GeneratePoolURL(urls.Etcd, "", polID),
	}
	dbRes := c.Get(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When get pool in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var pol = &model.StoragePoolSpec{}
	if err := json.Unmarshal([]byte(dbRes.Message[0]), pol); err != nil {
		log.Error("When parsing pool in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return pol, nil
}

//ListAvailabilityZones
func (c *Client) ListAvailabilityZones(ctx *c.Context) ([]string, error) {
	dbReq := &Request{
		Url: urls.GeneratePoolURL(urls.Etcd, ""),
	}
	dbRes := c.List(dbReq)
	if dbRes.Status != "Success" {
		log.Error("Failed to get AZ for pools in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	var azs = []string{}
	if len(dbRes.Message) == 0 {
		return azs, nil
	}
	for _, msg := range dbRes.Message {
		var pol = &model.StoragePoolSpec{}
		if err := json.Unmarshal([]byte(msg), pol); err != nil {
			log.Error("When parsing pool in db:", dbRes.Error)
			return nil, errors.New(dbRes.Error)
		}
		azs = append(azs, pol.AvailabilityZone)
	}
	//remove redundant AZ
	azs = utils.RvRepElement(azs)
	return azs, nil
}

// ListPools
func (c *Client) ListPools(ctx *c.Context) ([]*model.StoragePoolSpec, error) {
	dbReq := &Request{
		Url: urls.GeneratePoolURL(urls.Etcd, ""),
	}
	dbRes := c.List(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When list pools in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var pols = []*model.StoragePoolSpec{}
	if len(dbRes.Message) == 0 {
		return pols, nil
	}
	for _, msg := range dbRes.Message {
		var pol = &model.StoragePoolSpec{}
		if err := json.Unmarshal([]byte(msg), pol); err != nil {
			log.Error("When parsing pool in db:", dbRes.Error)
			return nil, errors.New(dbRes.Error)
		}
		pols = append(pols, pol)
	}
	return pols, nil
}

// UpdatePool
func (c *Client) UpdatePool(ctx *c.Context, polID, name, desp string, usedCapacity int64, used bool) (*model.StoragePoolSpec, error) {
	pol, err := c.GetPool(ctx, polID)
	if err != nil {
		return nil, err
	}
	if name != "" {
		pol.Name = name
	}
	if desp != "" {
		pol.Description = desp
	}
	pol.UpdatedAt = time.Now().Format(constants.TimeFormat)

	polBody, err := json.Marshal(pol)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:        urls.GeneratePoolURL(urls.Etcd, "", polID),
		NewContent: string(polBody),
	}
	dbRes := c.Update(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When update pool in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return pol, nil
}

// DeletePool
func (c *Client) DeletePool(ctx *c.Context, polID string) error {
	dbReq := &Request{
		Url: urls.GeneratePoolURL(urls.Etcd, "", polID),
	}
	dbRes := c.Delete(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When delete pool in db:", dbRes.Error)
		return errors.New(dbRes.Error)
	}
	return nil
}

// CreateProfile
func (c *Client) CreateProfile(ctx *c.Context, prf *model.ProfileSpec) (*model.ProfileSpec, error) {
	if prf.Id == "" {
		prf.Id = uuid.NewV4().String()
	}
	if prf.CreatedAt == "" {
		prf.CreatedAt = time.Now().Format(constants.TimeFormat)
	}

	// profile name must be unique.
	if _, err := c.getProfileByName(ctx, prf.Name); err == nil {
		return nil, fmt.Errorf("the profile name '%s' already exists", prf.Name)
	}

	prfBody, err := json.Marshal(prf)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:     urls.GenerateProfileURL(urls.Etcd, "", prf.Id),
		Content: string(prfBody),
	}
	dbRes := c.Create(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When create profile in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	return prf, nil
}

// GetProfile
func (c *Client) GetProfile(ctx *c.Context, prfID string) (*model.ProfileSpec, error) {
	dbReq := &Request{
		Url: urls.GenerateProfileURL(urls.Etcd, "", prfID),
	}
	dbRes := c.Get(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When get profile in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var prf = &model.ProfileSpec{}
	if err := json.Unmarshal([]byte(dbRes.Message[0]), prf); err != nil {
		log.Error("When parsing profile in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return prf, nil
}

func (c *Client) getProfileByName(ctx *c.Context, name string) (*model.ProfileSpec, error) {
	profiles, err := c.ListProfiles(ctx)
	if err != nil {
		log.Error("List profile failed: ", err)
		return nil, err
	}

	for _, profile := range profiles {
		if profile.Name == name {
			return profile, nil
		}
	}
	var msg = fmt.Sprintf("can't find profile(name: %s)", name)
	return nil, model.NewNotFoundError(msg)
}

func (c *Client) getProfileByNameAndType(ctx *c.Context, name, storageType string) (*model.ProfileSpec, error) {
	profiles, err := c.ListProfiles(ctx)
	if err != nil {
		log.Error("List profile failed: ", err)
		return nil, err
	}

	for _, profile := range profiles {
		if profile.Name == name && profile.StorageType == storageType {
			return profile, nil
		}
	}
	var msg = fmt.Sprintf("can't find profile(name: %s, storageType:%s)", name, storageType)
	return nil, model.NewNotFoundError(msg)
}

// GetDefaultProfile
func (c *Client) GetDefaultProfile(ctx *c.Context) (*model.ProfileSpec, error) {
	return c.getProfileByNameAndType(ctx, defaultBlockProfileName, typeBlock)
}

// GetDefaultProfileFileShare
func (c *Client) GetDefaultProfileFileShare(ctx *c.Context) (*model.ProfileSpec, error) {
	return c.getProfileByNameAndType(ctx, defaultFileProfileName, typeFile)
}

// ListProfiles
func (c *Client) ListProfiles(ctx *c.Context) ([]*model.ProfileSpec, error) {
	dbReq := &Request{
		Url: urls.GenerateProfileURL(urls.Etcd, ""),
	}

	dbRes := c.List(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When list profiles in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}

	var prfs = []*model.ProfileSpec{}
	if len(dbRes.Message) == 0 {
		return prfs, nil
	}
	for _, msg := range dbRes.Message {
		var prf = &model.ProfileSpec{}
		if err := json.Unmarshal([]byte(msg), prf); err != nil {
			log.Error("When parsing profile in db:", dbRes.Error)
			return nil, errors.New(dbRes.Error)
		}
		prfs = append(prfs, prf)
	}
	return prfs, nil
}

func (c *Client) ListProfilesWithFilter(ctx *c.Context, m map[string][]string) ([]*model.ProfileSpec, error) {
	profiles, err := c.ListProfiles(ctx)
	if err != nil {
		log.Error("List profiles failed: ", err)
		return nil, err
	}

	tmpProfiles := c.FilterAndSort(profiles, m, sortableKeysMap[typeProfiles])
	var res = []*model.ProfileSpec{}
	for _, data := range tmpProfiles.([]interface{}) {
		res = append(res, data.(*model.ProfileSpec))
	}
	return res, nil
}

// UpdateProfile
func (c *Client) UpdateProfile(ctx *c.Context, prfID string, input *model.ProfileSpec) (*model.ProfileSpec, error) {
	prf, err := c.GetProfile(ctx, prfID)
	if err != nil {
		return nil, err
	}
	if name := input.Name; name != "" {
		prf.Name = name
	}
	if desp := input.Description; desp != "" {
		prf.Description = desp
	}
	prf.UpdatedAt = time.Now().Format(constants.TimeFormat)

	if props := input.CustomProperties; len(props) != 0 {
		if prf.CustomProperties == nil {
			prf.CustomProperties = make(map[string]interface{})
		}
		for k, v := range props {
			prf.CustomProperties[k] = v
		}
	}

	prf.UpdatedAt = time.Now().Format(constants.TimeFormat)

	prfBody, err := json.Marshal(prf)
	if err != nil {
		return nil, err
	}

	dbReq := &Request{
		Url:        urls.GenerateProfileURL(urls.Etcd, "", prfID),
		NewContent: string(prfBody),
	}
	dbRes := c.Update(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When update profile in db:", dbRes.Error)
		return nil, errors.New(dbRes.Error)
	}
	return prf, nil
}

// DeleteProfile
func (c *Client) DeleteProfile(ctx *c.Context, prfID string) error {
	dbReq := &Request{
		Url: urls.GenerateProfileURL(urls.Etcd, "", prfID),
	}
	dbRes := c.Delete(dbReq)
	if dbRes.Status != "Success" {
		log.Error("When delete profile in db:", dbRes.Error)
		return errors.New(dbRes.Error)
	}
	return nil
}

// AddCustomProperty
func (c *Client) AddCustomProperty(ctx *c.Context, prfID string, ext model.CustomPropertiesSpec) (*model.CustomPropertiesSpec, error) {
	prf, err := c.GetProfile(ctx, prfID)
	if err != nil {
		return nil, err
	}

	if prf.CustomProperties == nil {
		prf.CustomProperties = make(map[string]interface{})
	}

	for k, v := range ext {
		prf.CustomProperties[k] = v
	}

	prf.UpdatedAt = time.Now().Format(constants.TimeFormat)

	if _, err = c.CreateProfile(ctx, prf); err != nil {
		return nil, err
	}
	return &prf.CustomProperties, nil
}

// ListCustomProperties
func (c *Client) ListCustomProperties(ctx *c.Context, prfID string) (*model.CustomPropertiesSpec, error) {
	prf, err := c.GetProfile(ctx, prfID)
	if err != nil {
		return nil, err
	}
	return &prf.CustomProperties, nil
}

// RemoveCustomProperty
func (c *Client) RemoveCustomProperty(ctx *c.Context, prfID, customKey string) error {
	prf, err := c.GetProfile(ctx, prfID)
	if err != nil {
		return err
	}

	delete(prf.CustomProperties, customKey)
	if _, err = c.CreateProfile(ctx, prf); err != nil {
		return err
	}
	return nil
}

func (c *Client) filterByName(param map[string][]string, spec interface{}, filterList map[string]interface{}) bool {
	v := reflect.ValueOf(spec)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for key := range param {
		_, ok := filterList[key]
		if !ok {
			continue
		}
		filed := v.FieldByName(key)
		if !filed.IsValid() {
			continue
		}
		paramVal := param[key][0]
		var val string
		switch filed.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val = strconv.FormatInt(filed.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val = strconv.FormatUint(filed.Uint(), 10)
		case reflect.String:
			val = filed.String()
		default:
			return false
		}
		if !strings.EqualFold(paramVal, val) {
			return false
		}
	}

	return true
}