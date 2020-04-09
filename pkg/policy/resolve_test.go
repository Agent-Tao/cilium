// Copyright 2018-2019 Authors of Cilium
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

// +build !privileged_tests

package policy

import (
	"fmt"
	"net"
	"sync"

	"github.com/cilium/cilium/pkg/checker"
	"github.com/cilium/cilium/pkg/identity"
	"github.com/cilium/cilium/pkg/identity/cache"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/option"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/cilium/cilium/pkg/policy/trafficdirection"
	"github.com/cilium/cilium/pkg/testutils/allocator"
	. "gopkg.in/check.v1"
)

var (
	fooLabel = labels.NewLabel("k8s:foo", "", "")
	lbls     = labels.Labels{
		"foo": fooLabel,
	}
	lblsArray   = lbls.LabelArray()
	fooIdentity = &identity.Identity{
		ID:         303,
		Labels:     lbls,
		LabelArray: lbls.LabelArray(),
	}
	identityCache = cache.IdentityCache{303: lblsArray}
)

type dummyEndpoint struct {
	ID               uint16
	SecurityIdentity *identity.Identity
}

func (d *dummyEndpoint) GetID16() uint16 {
	return d.ID
}

func (d *dummyEndpoint) GetSecurityIdentity() (*identity.Identity, error) {
	return d.SecurityIdentity, nil
}

func (d *dummyEndpoint) PolicyRevisionBumpEvent(rev uint64) {
}

func GenerateNumIdentities(numIdentities int) {
	for i := 0; i < numIdentities; i++ {

		identityLabel := labels.NewLabel(fmt.Sprintf("k8s:foo%d", i), "", "")
		clusterLabel := labels.NewLabel("io.cilium.k8s.policy.cluster=default", "", labels.LabelSourceK8s)
		serviceAccountLabel := labels.NewLabel("io.cilium.k8s.policy.serviceaccount=default", "", labels.LabelSourceK8s)
		namespaceLabel := labels.NewLabel("io.kubernetes.pod.namespace=monitoring", "", labels.LabelSourceK8s)
		funLabel := labels.NewLabel("app=analytics-erneh", "", labels.LabelSourceK8s)

		identityLabels := labels.Labels{
			fmt.Sprintf("foo%d", i):                           identityLabel,
			"k8s:io.cilium.k8s.policy.cluster=default":        clusterLabel,
			"k8s:io.cilium.k8s.policy.serviceaccount=default": serviceAccountLabel,
			"k8s:io.kubernetes.pod.namespace=monitoring":      namespaceLabel,
			"k8s:app=analytics-erneh":                         funLabel,
		}

		bumpedIdentity := i + 1000
		numericIdentity := identity.NumericIdentity(bumpedIdentity)

		identityCache[numericIdentity] = identityLabels.LabelArray()
	}
}

func GenerateL3IngressRules(numRules int) api.Rules {
	parseFooLabel := labels.ParseSelectLabel("k8s:foo")
	fooSelector := api.NewESFromLabels(parseFooLabel)
	barSelector := api.NewESFromLabels(labels.ParseSelectLabel("bar"))

	// Change ingRule and rule in the for-loop below to change what type of rules
	// are added into the policy repository.
	ingRule := api.IngressRule{
		FromEndpoints: []api.EndpointSelector{barSelector},
	}

	var rules api.Rules
	for i := 1; i <= numRules; i++ {

		rule := api.Rule{
			EndpointSelector: fooSelector,
			Ingress:          []api.IngressRule{ingRule},
		}
		rule.Sanitize()
		rules = append(rules, &rule)
	}
	return rules
}

func GenerateL3EgressRules(numRules int) api.Rules {
	parseFooLabel := labels.ParseSelectLabel("k8s:foo")
	fooSelector := api.NewESFromLabels(parseFooLabel)
	barSelector := api.NewESFromLabels(labels.ParseSelectLabel("bar"))

	// Change ingRule and rule in the for-loop below to change what type of rules
	// are added into the policy repository.
	egRule := api.EgressRule{
		ToEndpoints: []api.EndpointSelector{barSelector},
	}

	var rules api.Rules
	for i := 1; i <= numRules; i++ {

		rule := api.Rule{
			EndpointSelector: fooSelector,
			Egress:           []api.EgressRule{egRule},
		}
		rule.Sanitize()
		rules = append(rules, &rule)
	}
	return rules
}

func GenerateL3EgressFQDNRules(numRules int) api.Rules {
	parseFooLabel := labels.ParseSelectLabel("k8s:foo")
	fooSelector := api.NewESFromLabels(parseFooLabel)

	barSelector := api.FQDNSelector{MatchPattern: "*.BAR.COM"}
	err := barSelector.Sanitize()
	if err != nil {
		panic(err)
	}

	// Change ingRule and rule in the for-loop below to change what type of rules
	// are added into the policy repository.
	egRule := api.EgressRule{
		ToFQDNs: []api.FQDNSelector{barSelector},
	}

	var rules api.Rules
	for i := 1; i <= numRules; i++ {
		rule := api.Rule{
			EndpointSelector: fooSelector,
			Egress:           []api.EgressRule{egRule},
		}
		rule.Sanitize()
		rules = append(rules, &rule)
	}
	return rules
}

func GenerateCIDRRules(numRules int) api.Rules {
	parseFooLabel := labels.ParseSelectLabel("k8s:foo")
	fooSelector := api.NewESFromLabels(parseFooLabel)
	//barSelector := api.NewESFromLabels(labels.ParseSelectLabel("bar"))

	// Change ingRule and rule in the for-loop below to change what type of rules
	// are added into the policy repository.
	egRule := api.EgressRule{
		ToCIDR: []api.CIDR{api.CIDR("10.2.3.0/24"), api.CIDR("ff02::/64")},
		/*ToRequires:  []api.EndpointSelector{barSelector},
		ToPorts: []api.PortRule{
			{
				Ports: []api.PortProtocol{
					{
						Port:     "8080",
						Protocol: api.ProtoTCP,
					},
				},
			},
		},*/
	}

	var rules api.Rules
	for i := 1; i <= numRules; i++ {

		rule := api.Rule{
			EndpointSelector: fooSelector,
			Egress:           []api.EgressRule{egRule},
		}
		rule.Sanitize()
		rules = append(rules, &rule)
	}
	return rules
}

type DummyOwner struct{}

func (d DummyOwner) LookupRedirectPort(l4 *L4Filter) uint16 {
	// Return a fake non-0 listening port number for redirect filters.
	if l4.IsRedirect() {
		return 4242
	}
	return 0
}

type MockDNSProxy struct {
	allSelectors map[api.FQDNSelector]struct{}
	dnsdata      map[string][]net.IP
}

var mockDNSProxy MockDNSProxy = MockDNSProxy{
	allSelectors: map[api.FQDNSelector]struct{}{},
	dnsdata: map[string][]net.IP{
		"foo.bar.com.": {net.ParseIP("1.1.1.1"), net.ParseIP("1.1.1.2")},
		"bar.bar.com.": {net.ParseIP("2.2.2.1"), net.ParseIP("2.2.2.2")},
		"bar.foo.com.": {net.ParseIP("3.3.3.1"), net.ParseIP("3.3.3.2")},
	},
}

// Lock must be held during any calls to RegisterForIdentityUpdatesLocked or
// UnregisterForIdentityUpdatesLocked.
func (d *MockDNSProxy) Lock() {}

// Unlock must be called after calls to RegisterForIdentityUpdatesLocked or
// UnregisterForIdentityUpdatesLocked are done.
func (d *MockDNSProxy) Unlock() {}

// RegisterForIdentityUpdatesLocked exposes this FQDNSelector so that identities
// for IPs contained in a DNS response that matches said selector can be
// propagated back to the SelectorCache via `UpdateFQDNSelector`. When called,
// implementers (currently `pkg/fqdn/RuleGen`) should iterate over all DNS
// names that they are aware of, and see if they match the FQDNSelector.
// All IPs which correspond to the DNS names which match this Selector will
// be returned as CIDR identities, as other DNS Names which have already
// been resolved may match this FQDNSelector.
// Once this function is called, the SelectorCache will be updated any time
// new IPs are resolved for DNS names which match this FQDNSelector.
// This function is only called when the SelectorCache has been made aware
// of this FQDNSelector for the first time, since we only need to get the
// set of CIDR identities which match this FQDNSelector already from the
// identityNotifier on the first pass; any subsequent updates will eventually
// call `UpdateFQDNSelector`.
func (d *MockDNSProxy) RegisterForIdentityUpdatesLocked(selector api.FQDNSelector) (identities []identity.NumericIdentity) {
	re := selector.ToRegex()
	var ips []net.IP
	for k, v := range d.dnsdata {
		if re.MatchString(k) {
			ips = append(ips, v...)
		}
	}

	// to be completed

	return nil
}

// UnregisterForIdentityUpdatesLocked removes this FQDNSelector from the set of
// FQDNSelectors which are being tracked by the identityNotifier. The result
// of this is that no more updates for IPs which correspond to said selector
// are propagated back to the SelectorCache via `UpdateFQDNSelector`.
// This occurs when there are no more users of a given FQDNSelector for the
// SelectorCache.
func (d *MockDNSProxy) UnregisterForIdentityUpdatesLocked(selector api.FQDNSelector) {
}

func bootstrapRepo(ruleGenFunc func(int) api.Rules, numRules int, c *C) *Repository {
	mgr := cache.NewCachingIdentityAllocator(&allocator.IdentityAllocatorOwnerMock{})
	testRepo := NewPolicyRepository(mgr.GetIdentityCache(), nil)

	var wg sync.WaitGroup
	SetPolicyEnabled(option.DefaultEnforcement)
	GenerateNumIdentities(3000)
	testSelectorCache.UpdateIdentities(identityCache, nil)
	testSelectorCache.SetLocalIdentityNotifier(&mockDNSProxy)
	testRepo.selectorCache = testSelectorCache
	rulez, _ := testRepo.AddList(ruleGenFunc(numRules))

	epSet := NewEndpointSet(map[Endpoint]struct{}{
		&dummyEndpoint{
			ID:               9001,
			SecurityIdentity: fooIdentity,
		}: {},
	})

	epsToRegen := NewEndpointSet(nil)
	rulez.UpdateRulesEndpointsCaches(epSet, epsToRegen, &wg)
	wg.Wait()

	c.Assert(epSet.Len(), Equals, 0)
	c.Assert(epsToRegen.Len(), Equals, 1)

	return testRepo
}

func (ds *PolicyTestSuite) BenchmarkRegenerateCIDRPolicyRules(c *C) {
	testRepo := bootstrapRepo(GenerateCIDRRules, 1000, c)

	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		ip, _ := testRepo.resolvePolicyLocked(fooIdentity)
		_ = ip.DistillPolicy(DummyOwner{})
		ip.Detach()
	}
}

func (ds *PolicyTestSuite) BenchmarkRegenerateL3IngressPolicyRules(c *C) {
	testRepo := bootstrapRepo(GenerateL3IngressRules, 1000, c)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		ip, _ := testRepo.resolvePolicyLocked(fooIdentity)
		_ = ip.DistillPolicy(DummyOwner{})
		ip.Detach()
	}
}

func (ds *PolicyTestSuite) BenchmarkRegenerateL3EgressPolicyRules(c *C) {
	testRepo := bootstrapRepo(GenerateL3EgressRules, 1000, c)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		ip, _ := testRepo.resolvePolicyLocked(fooIdentity)
		_ = ip.DistillPolicy(DummyOwner{})
		ip.Detach()
	}
}

func (ds *PolicyTestSuite) BenchmarkRegenerateL3EgressFQDNPolicyRules(c *C) {
	testRepo := bootstrapRepo(GenerateL3EgressFQDNRules, 1000, c)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		ip, _ := testRepo.resolvePolicyLocked(fooIdentity)
		_ = ip.DistillPolicy(DummyOwner{})
		ip.Detach()
	}
}

func (ds *PolicyTestSuite) TestL7WithIngressWildcard(c *C) {
	repo := bootstrapRepo(GenerateL3IngressRules, 1000, c)

	idFooSelectLabelArray := labels.ParseSelectLabelArray("id=foo")
	idFooSelectLabels := labels.Labels{}
	for _, lbl := range idFooSelectLabelArray {
		idFooSelectLabels[lbl.Key] = lbl
	}
	fooIdentity := identity.NewIdentity(12345, idFooSelectLabels)

	selFoo := api.NewESFromLabels(labels.ParseSelectLabel("id=foo"))
	rule1 := api.Rule{
		EndpointSelector: selFoo,
		Ingress: []api.IngressRule{
			{
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{
						{Port: "80", Protocol: api.ProtoTCP},
					},
					Rules: &api.L7Rules{
						HTTP: []api.PortRuleHTTP{
							{Method: "GET", Path: "/good"},
						},
					},
				}},
			},
		},
	}

	rule1.Sanitize()
	_, _, err := repo.Add(rule1, []Endpoint{})
	c.Assert(err, IsNil)

	repo.Mutex.RLock()
	defer repo.Mutex.RUnlock()
	selPolicy, err := repo.resolvePolicyLocked(fooIdentity)
	c.Assert(err, IsNil)
	policy := selPolicy.DistillPolicy(DummyOwner{})

	expectedEndpointPolicy := EndpointPolicy{
		selectorPolicy: &selectorPolicy{
			Revision:      repo.GetRevision(),
			SelectorCache: repo.GetSelectorCache(),
			L4Policy: &L4Policy{
				Revision: repo.GetRevision(),
				Ingress: L4PolicyMap{
					"80/TCP": {
						Port:     80,
						Protocol: api.ProtoTCP,
						U8Proto:  0x6,
						wildcard: wildcardCachedSelector,
						L7Parser: ParserTypeHTTP,
						Ingress:  true,
						L7RulesPerSelector: L7DataMap{
							wildcardCachedSelector: &PerSelectorPolicy{
								L7Rules: api.L7Rules{
									HTTP: []api.PortRuleHTTP{{Method: "GET", Path: "/good"}},
								},
								CanShortCircuit: true,
							},
						},
						DerivedFromRules: labels.LabelArrayList{nil},
					},
				},
				Egress: L4PolicyMap{},
			},
			CIDRPolicy:           policy.CIDRPolicy,
			IngressPolicyEnabled: true,
			EgressPolicyEnabled:  false,
		},
		PolicyOwner: DummyOwner{},
		// inherit this from the result as it is outside of the scope
		// of this test
		PolicyMapState: policy.PolicyMapState,
	}

	// Have to remove circular reference before testing to avoid an infinite loop
	policy.selectorPolicy.Detach()

	c.Assert(policy, checker.Equals, &expectedEndpointPolicy)
}

func (ds *PolicyTestSuite) TestL7WithLocalHostWildcardd(c *C) {
	repo := bootstrapRepo(GenerateL3IngressRules, 1000, c)

	idFooSelectLabelArray := labels.ParseSelectLabelArray("id=foo")
	idFooSelectLabels := labels.Labels{}
	for _, lbl := range idFooSelectLabelArray {
		idFooSelectLabels[lbl.Key] = lbl
	}

	fooIdentity := identity.NewIdentity(12345, idFooSelectLabels)

	// Emulate Kubernetes mode with allow from localhost
	oldLocalhostOpt := option.Config.AllowLocalhost
	option.Config.AllowLocalhost = option.AllowLocalhostAlways
	defer func() { option.Config.AllowLocalhost = oldLocalhostOpt }()

	selFoo := api.NewESFromLabels(labels.ParseSelectLabel("id=foo"))
	rule1 := api.Rule{
		EndpointSelector: selFoo,
		Ingress: []api.IngressRule{
			{
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{
						{Port: "80", Protocol: api.ProtoTCP},
					},
					Rules: &api.L7Rules{
						HTTP: []api.PortRuleHTTP{
							{Method: "GET", Path: "/good"},
						},
					},
				}},
			},
		},
	}

	rule1.Sanitize()
	_, _, err := repo.Add(rule1, []Endpoint{})
	c.Assert(err, IsNil)

	repo.Mutex.RLock()
	defer repo.Mutex.RUnlock()

	selPolicy, err := repo.resolvePolicyLocked(fooIdentity)
	c.Assert(err, IsNil)
	policy := selPolicy.DistillPolicy(DummyOwner{})

	cachedSelectorHost := testSelectorCache.FindCachedIdentitySelector(api.ReservedEndpointSelectors[labels.IDNameHost])
	c.Assert(cachedSelectorHost, Not(IsNil))

	expectedEndpointPolicy := EndpointPolicy{
		selectorPolicy: &selectorPolicy{
			Revision:      repo.GetRevision(),
			SelectorCache: repo.GetSelectorCache(),
			L4Policy: &L4Policy{
				Revision: repo.GetRevision(),
				Ingress: L4PolicyMap{
					"80/TCP": {
						Port:     80,
						Protocol: api.ProtoTCP,
						U8Proto:  0x6,
						wildcard: wildcardCachedSelector,
						L7Parser: ParserTypeHTTP,
						Ingress:  true,
						L7RulesPerSelector: L7DataMap{
							wildcardCachedSelector: &PerSelectorPolicy{
								L7Rules: api.L7Rules{
									HTTP: []api.PortRuleHTTP{{Method: "GET", Path: "/good"}},
								},
								CanShortCircuit: true,
							},
							cachedSelectorHost: nil,
						},
						DerivedFromRules: labels.LabelArrayList{nil},
					},
				},
				Egress: L4PolicyMap{},
			},
			CIDRPolicy:           policy.CIDRPolicy,
			IngressPolicyEnabled: true,
			EgressPolicyEnabled:  false,
		},
		PolicyOwner: DummyOwner{},
		// inherit this from the result as it is outside of the scope
		// of this test
		PolicyMapState: policy.PolicyMapState,
	}

	// Have to remove circular reference before testing to avoid an infinite loop
	policy.selectorPolicy.Detach()

	c.Assert(policy, checker.Equals, &expectedEndpointPolicy)
}

func (ds *PolicyTestSuite) TestMapStateWithIngressWildcard(c *C) {
	repo := bootstrapRepo(GenerateL3IngressRules, 1000, c)

	ruleLabel := labels.ParseLabelArray("rule-foo-allow-port-80")
	ruleLabelAllowAnyEgress := labels.LabelArray{
		labels.NewLabel(LabelKeyPolicyDerivedFrom, LabelAllowAnyEgress, labels.LabelSourceReserved),
	}

	idFooSelectLabelArray := labels.ParseSelectLabelArray("id=foo")
	idFooSelectLabels := labels.Labels{}
	for _, lbl := range idFooSelectLabelArray {
		idFooSelectLabels[lbl.Key] = lbl
	}
	fooIdentity := identity.NewIdentity(12345, idFooSelectLabels)

	selFoo := api.NewESFromLabels(labels.ParseSelectLabel("id=foo"))
	rule1 := api.Rule{
		EndpointSelector: selFoo,
		Labels:           ruleLabel,
		Ingress: []api.IngressRule{
			{
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{
						{Port: "80", Protocol: api.ProtoTCP},
					},
					Rules: &api.L7Rules{},
				}},
			},
		},
	}

	rule1.Sanitize()
	_, _, err := repo.Add(rule1, []Endpoint{})
	c.Assert(err, IsNil)

	repo.Mutex.RLock()
	defer repo.Mutex.RUnlock()
	selPolicy, err := repo.resolvePolicyLocked(fooIdentity)
	c.Assert(err, IsNil)
	policy := selPolicy.DistillPolicy(DummyOwner{})

	rule1MapStateEntry := NewMapStateEntry(labels.LabelArrayList{ruleLabel}, false)
	allowEgressMapStateEntry := NewMapStateEntry(labels.LabelArrayList{ruleLabelAllowAnyEgress}, false)

	expectedEndpointPolicy := EndpointPolicy{
		selectorPolicy: &selectorPolicy{
			Revision:      repo.GetRevision(),
			SelectorCache: repo.GetSelectorCache(),
			L4Policy: &L4Policy{
				Revision: repo.GetRevision(),
				Ingress: L4PolicyMap{
					"80/TCP": {
						Port:     80,
						Protocol: api.ProtoTCP,
						U8Proto:  0x6,
						wildcard: wildcardCachedSelector,
						L7Parser: ParserTypeNone,
						Ingress:  true,
						L7RulesPerSelector: L7DataMap{
							wildcardCachedSelector: nil,
						},
						DerivedFromRules: labels.LabelArrayList{ruleLabel},
					},
				},
				Egress: L4PolicyMap{},
			},
			CIDRPolicy:           policy.CIDRPolicy,
			IngressPolicyEnabled: true,
			EgressPolicyEnabled:  false,
		},
		PolicyOwner: DummyOwner{},
		PolicyMapState: MapState{
			{TrafficDirection: trafficdirection.Egress.Uint8()}: allowEgressMapStateEntry,
			{DestPort: 80, Nexthdr: 6}:                          rule1MapStateEntry,
		},
	}

	// Add new identity to test accumulation of PolicyMapChanges
	added1 := cache.IdentityCache{
		identity.NumericIdentity(192): labels.ParseSelectLabelArray("id=resolve_test_1"),
	}
	testSelectorCache.UpdateIdentities(added1, nil)
	c.Assert(policy.policyMapChanges.adds, HasLen, 0)
	c.Assert(policy.policyMapChanges.deletes, HasLen, 0)

	// Have to remove circular reference before testing to avoid an infinite loop
	policy.selectorPolicy.Detach()

	c.Assert(policy, checker.Equals, &expectedEndpointPolicy)
}

func (ds *PolicyTestSuite) TestMapStateWithIngress(c *C) {
	repo := bootstrapRepo(GenerateL3IngressRules, 1000, c)

	ruleLabel := labels.ParseLabelArray("rule-world-allow-port-80")
	ruleLabelAllowAnyEgress := labels.LabelArray{
		labels.NewLabel(LabelKeyPolicyDerivedFrom, LabelAllowAnyEgress, labels.LabelSourceReserved),
	}

	idFooSelectLabelArray := labels.ParseSelectLabelArray("id=foo")
	idFooSelectLabels := labels.Labels{}
	for _, lbl := range idFooSelectLabelArray {
		idFooSelectLabels[lbl.Key] = lbl
	}
	fooIdentity := identity.NewIdentity(12345, idFooSelectLabels)

	lblTest := labels.ParseLabel("id=resolve_test_1")

	selFoo := api.NewESFromLabels(labels.ParseSelectLabel("id=foo"))
	rule1 := api.Rule{
		EndpointSelector: selFoo,
		Labels:           ruleLabel,
		Ingress: []api.IngressRule{
			{
				FromEntities: []api.Entity{api.EntityWorld},
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{
						{Port: "80", Protocol: api.ProtoTCP},
					},
					Rules: &api.L7Rules{},
				}},
			},
			{
				FromEndpoints: []api.EndpointSelector{
					api.NewESFromLabels(lblTest),
				},
				ToPorts: []api.PortRule{{
					Ports: []api.PortProtocol{
						{Port: "80", Protocol: api.ProtoTCP},
					},
					Rules: &api.L7Rules{},
				}},
			},
		},
	}

	rule1.Sanitize()
	_, _, err := repo.Add(rule1, []Endpoint{})
	c.Assert(err, IsNil)

	repo.Mutex.RLock()
	defer repo.Mutex.RUnlock()
	selPolicy, err := repo.resolvePolicyLocked(fooIdentity)
	c.Assert(err, IsNil)
	policy := selPolicy.DistillPolicy(DummyOwner{})

	// Add new identity to test accumulation of PolicyMapChanges
	added1 := cache.IdentityCache{
		identity.NumericIdentity(192): labels.ParseSelectLabelArray("id=resolve_test_1", "num=1"),
		identity.NumericIdentity(193): labels.ParseSelectLabelArray("id=resolve_test_1", "num=2"),
		identity.NumericIdentity(194): labels.ParseSelectLabelArray("id=resolve_test_1", "num=3"),
	}
	testSelectorCache.UpdateIdentities(added1, nil)
	c.Assert(policy.policyMapChanges.adds, HasLen, 3)
	c.Assert(policy.policyMapChanges.deletes, HasLen, 0)

	deleted1 := cache.IdentityCache{
		identity.NumericIdentity(193): labels.ParseSelectLabelArray("id=resolve_test_1", "num=2"),
	}
	testSelectorCache.UpdateIdentities(nil, deleted1)
	c.Assert(policy.policyMapChanges.adds, HasLen, 2)
	c.Assert(policy.policyMapChanges.deletes, HasLen, 1)

	cachedSelectorWorld := testSelectorCache.FindCachedIdentitySelector(api.ReservedEndpointSelectors[labels.IDNameWorld])
	c.Assert(cachedSelectorWorld, Not(IsNil))

	cachedSelectorTest := testSelectorCache.FindCachedIdentitySelector(api.NewESFromLabels(lblTest))
	c.Assert(cachedSelectorTest, Not(IsNil))

	rule1MapStateEntry := NewMapStateEntry(labels.LabelArrayList{ruleLabel, ruleLabel}, false)
	allowEgressMapStateEntry := NewMapStateEntry(labels.LabelArrayList{ruleLabelAllowAnyEgress}, false)

	expectedEndpointPolicy := EndpointPolicy{
		selectorPolicy: &selectorPolicy{
			Revision:      repo.GetRevision(),
			SelectorCache: repo.GetSelectorCache(),
			L4Policy: &L4Policy{
				Revision: repo.GetRevision(),
				Ingress: L4PolicyMap{
					"80/TCP": {
						Port:     80,
						Protocol: api.ProtoTCP,
						U8Proto:  0x6,
						L7Parser: ParserTypeNone,
						Ingress:  true,
						L7RulesPerSelector: L7DataMap{
							cachedSelectorWorld: nil,
							cachedSelectorTest:  nil,
						},
						DerivedFromRules: labels.LabelArrayList{ruleLabel, ruleLabel},
					},
				},
				Egress: L4PolicyMap{},
			},
			CIDRPolicy:           policy.CIDRPolicy,
			IngressPolicyEnabled: true,
			EgressPolicyEnabled:  false,
		},
		PolicyOwner: DummyOwner{},
		PolicyMapState: MapState{
			{TrafficDirection: trafficdirection.Egress.Uint8()}:                          allowEgressMapStateEntry,
			{Identity: uint32(identity.ReservedIdentityWorld), DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
		},
		policyMapChanges: MapChanges{
			adds: MapState{
				{Identity: 192, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
				{Identity: 194, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
			},
			deletes: MapState{
				{Identity: 193, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
			},
		},
	}

	// Have to remove circular reference before testing for Equality to avoid an infinite loop
	policy.selectorPolicy.Detach()
	// Verify that cached selector is not found after Detach().
	// Note that this depends on the other tests NOT using the same selector concurrently!
	cachedSelectorTest = testSelectorCache.FindCachedIdentitySelector(api.NewESFromLabels(lblTest))
	c.Assert(cachedSelectorTest, IsNil)

	c.Assert(policy, checker.Equals, &expectedEndpointPolicy)

	adds, deletes := policy.ConsumeMapChanges()
	// maps on the policy got cleared
	c.Assert(policy.policyMapChanges.adds, IsNil)
	c.Assert(policy.policyMapChanges.deletes, IsNil)

	c.Assert(adds, checker.Equals, MapState{
		{Identity: 192, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
		{Identity: 194, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
	})
	c.Assert(deletes, checker.Equals, MapState{
		{Identity: 193, DestPort: 80, Nexthdr: 6}: rule1MapStateEntry,
	})
}
