package reconcile

import (
	"context"
	"sync"
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ReconcileSuite struct {
	cli       kubernetes.Interface
	crCli     crclientv1alpha1.CrV1alpha1Interface
	namespace string
	as        *crv1alpha1.ActionSet
}

var _ = Suite(&ReconcileSuite{})

const (
	namespace = "default"
)

func (s *ReconcileSuite) SetUpSuite(c *C) {
	// Setup Clients
	config, err := kube.LoadConfig()
	c.Assert(err, IsNil)
	cli, err := kubernetes.NewForConfig(config)
	c.Assert(err, IsNil)
	s.cli = cli

	crCli, err := crclientv1alpha1.NewForConfig(config)
	c.Assert(err, IsNil)
	s.crCli = crCli

	// Create Namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "reconciletest-",
		},
	}
	cns, err := cli.Core().Namespaces().Create(ns)
	c.Assert(err, IsNil)
	s.namespace = cns.Name

	// Create ActionSet
	as := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "reconciletest-",
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: []crv1alpha1.ActionSpec{
				crv1alpha1.ActionSpec{},
			},
		},
		Status: &crv1alpha1.ActionSetStatus{
			Actions: []crv1alpha1.ActionStatus{
				crv1alpha1.ActionStatus{
					Phases: []crv1alpha1.Phase{
						crv1alpha1.Phase{
							State: crv1alpha1.StatePending,
						},
						crv1alpha1.Phase{
							State: crv1alpha1.StatePending,
						},
					},
				},
			},
			State: crv1alpha1.StatePending,
		},
	}
	as, err = s.crCli.ActionSets(s.namespace).Create(as)
	c.Assert(err, IsNil)
	s.as = as
}

func (s *ReconcileSuite) TearDownSuite(c *C) {
	if s.namespace != "" {
		s.cli.Core().Namespaces().Delete(s.namespace, nil)
	}
}

func (s *ReconcileSuite) TestSetFailed(c *C) {
	ctx := context.Background()
	err := ActionSet(ctx, s.crCli, s.namespace, s.as.GetName(), func(as *crv1alpha1.ActionSet) error {
		as.Status.Actions[0].Phases[0].State = crv1alpha1.StateFailed
		as.Status.State = crv1alpha1.StateFailed
		return nil
	})
	c.Assert(err, IsNil)

	as, err := s.crCli.ActionSets(s.namespace).Get(s.as.GetName(), metav1.GetOptions{})
	c.Assert(err, IsNil)
	c.Assert(as.Status.State, Equals, crv1alpha1.StateFailed)
}

// Tested with 30, but it took 20 seconds to run. This takes 2 seconds and we
// still see conflicts.
const numParallel = 5

func (s *ReconcileSuite) TestHandleConflict(c *C) {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	for range make([]struct{}, numParallel) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ActionSet(ctx, s.crCli, s.namespace, s.as.GetName(), func(as *crv1alpha1.ActionSet) error {
				as.Status.Actions[0].Phases[0].State = crv1alpha1.StateFailed
				as.Status.State = crv1alpha1.StateFailed
				return nil
			})
			c.Assert(err, IsNil)
		}()
	}
	wg.Wait()
}