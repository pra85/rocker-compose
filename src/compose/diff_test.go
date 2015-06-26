package compose

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"fmt"
)

func TestComparatorSameValue(t *testing.T) {
	cmp := NewDiff()
	containers := make([]*Container, 0)
	act, err := cmp.Diff("", containers, containers)
	assert.Empty(t, act)
	assert.Nil(t, err)
}

func TestDiffCreateAll(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1", ContainerName{"test", "2"}, ContainerName{"test", "3"})
	c2 := newContainer("test", "2", ContainerName{"test", "4"})
	c3 := newContainer("test", "3", ContainerName{"test", "4"})
	c4 := newContainer("test", "4")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff("test", containers, []*Container{})
	mock := clientMock{}
	mock.On("CreateContainer", c4).Return(nil)
	mock.On("CreateContainer", c2).Return(nil)
	mock.On("CreateContainer", c3).Return(nil)
	mock.On("CreateContainer", c1).Return(nil)

	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffNoDependencies(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	containers = append(containers, c1, c2, c3)
	actions, _ := cmp.Diff("test", containers, []*Container{})
	mock := clientMock{}
	mock.On("CreateContainer", c1).Return(nil)
	mock.On("CreateContainer", c2).Return(nil)
	mock.On("CreateContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffAddingOneContainer(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	containers = append(containers, c1, c2)
	actions, _ := cmp.Diff("test", containers, []*Container{c1, c3})
	mock := clientMock{}
	mock.On("CreateContainer", c2).Return(nil)
	mock.On("RemoveContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffExternalDependencies(t *testing.T) {
	cmp := NewDiff()
	c1 := newContainer("metrics", "1")
	c2 := newContainer("metrics", "2")
	c3 := newContainer("metrics", "3")
	actions, _ := cmp.Diff("test", []*Container{}, []*Container{c1, c2, c3})
	mock := clientMock{}
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffEnsureFewExternalDependencies(t *testing.T) {
	cmp := NewDiff()
	c1 := newContainer("metrics", "1")
	c2 := newContainer("metrics", "2")
	c3 := newContainer("metrics", "3")
	c4 := newContainer("test", "1", ContainerName{"metrics", "1"},
		ContainerName{"metrics", "2"}, ContainerName{"metrics", "3"})
	actions, _ := cmp.Diff("test", []*Container{c4}, []*Container{c1, c2, c3})
	mock := clientMock{}
	mock.On("EnsureContainer", c1).Return(nil)
	mock.On("EnsureContainer", c2).Return(nil)
	mock.On("EnsureContainer", c3).Return(nil)
	mock.On("CreateContainer", c4).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffFailInMiddle(t *testing.T) {
	cmp := NewDiff()
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3")
	actions, _ := cmp.Diff("test", []*Container{c1, c2, c3}, []*Container{})
	mock := clientMock{}
	mock.On("CreateContainer", c1).Return(nil)
	mock.On("CreateContainer", c2).Return(fmt.Errorf("fail"))
	mock.On("CreateContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	assert.Error(t, runner.Run(actions))
	mock.AssertExpectations(t)
}

func TestDiffFailInDependent(t *testing.T) {
	cmp := NewDiff()
	c1 := newContainer("test", "1", ContainerName{"test", "2"})
	c2 := newContainer("test", "2")
	c3 := newContainer("test", "3", ContainerName{"test", "2"})
	actions, _ := cmp.Diff("test", []*Container{c1, c2, c3}, []*Container{})
	mock := clientMock{}
	mock.On("CreateContainer", c2).Return(fmt.Errorf("fail"))
	runner := NewDockerClientRunner(&mock)
	assert.Error(t, runner.Run(actions))
	mock.AssertExpectations(t)
}

func TestDiffForCycles(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1", ContainerName{"test", "2"})
	c2 := newContainer("test", "2", ContainerName{"test", "3"})
	c3 := newContainer("test", "3", ContainerName{"test", "1"})
	containers = append(containers, c1, c2, c3)
	_, err := cmp.Diff("test", containers, []*Container{c1, c3})
	assert.Error(t, err)
}

func TestDiffDifferentConfig(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1x := &Container{
		State: &ContainerState{Running: true},
		Name: &ContainerName{"test", "1"},
		Config: &ConfigContainer{CpusetCpus:"difference"},
	}
	c1y := newContainer("test", "1")
	containers = append(containers, c1x)
	actions, _ := cmp.Diff("test", containers, []*Container{c1y})
	mock := clientMock{}
	mock.On("RemoveContainer", c1y).Return(nil)
	mock.On("CreateContainer", c1x).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffForExternalDependencies(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1")
	c2 := newContainer("test", "2", ContainerName{"metrics", "1"})
	m1 := newContainer("metrics", "1")
	containers = append(containers, c1, c2)
	actions, _ := cmp.Diff("test", containers, []*Container{m1})
	mock := clientMock{}
	mock.On("EnsureContainer", m1).Return(nil)
	mock.On("CreateContainer", c1).Return(nil)
	mock.On("CreateContainer", c2).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffCreateRemoving(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1", ContainerName{"test", "2"}, ContainerName{"test", "3"})
	c2 := newContainer("test", "2", ContainerName{"test", "4"})
	c3 := newContainer("test", "3", ContainerName{"test", "4"})
	c4 := newContainer("test", "4")
	c5 := newContainer("test", "5")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff("test", containers, []*Container{c5})
	mock := clientMock{}
	mock.On("RemoveContainer", c5).Return(nil)
	mock.On("CreateContainer", c4).Return(nil)
	mock.On("CreateContainer", c2).Return(nil)
	mock.On("CreateContainer", c3).Return(nil)
	mock.On("CreateContainer", c1).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func TestDiffCreateSome(t *testing.T) {
	cmp := NewDiff()
	containers := []*Container{}
	c1 := newContainer("test", "1", ContainerName{"test", "2"}, ContainerName{"test", "3"})
	c2 := newContainer("test", "2", ContainerName{"test", "4"})
	c3 := newContainer("test", "3", ContainerName{"test", "4"})
	c4 := newContainer("test", "4")
	containers = append(containers, c1, c2, c3, c4)
	actions, _ := cmp.Diff("test", containers, []*Container{c1})
	mock := clientMock{}
	mock.On("CreateContainer", c4).Return(nil)
	mock.On("CreateContainer", c2).Return(nil)
	mock.On("CreateContainer", c3).Return(nil)
	runner := NewDockerClientRunner(&mock)
	runner.Run(actions)
	mock.AssertExpectations(t)
}

func newContainer(namespace string, name string, dependencies ...ContainerName) *Container {
	return &Container{
		State: &ContainerState{
			Running: true,
		},
		Name: &ContainerName{namespace, name},
		Config: &ConfigContainer{
			VolumesFrom: dependencies,
		}}
}

func (m *clientMock) GetContainers() ([]*Container, error) {
	args := m.Called()
	return nil, args.Error(0)
}

func (m *clientMock) RemoveContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) CreateContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) EnsureContainer(container *Container) error {
	args := m.Called(container)
	return args.Error(0)
}

func (m *clientMock) PullImage(imageName *ImageName) error {
	args := m.Called(imageName)
	return args.Error(0)
}

func (m *clientMock) PullAll(config *Config) error {
	args := m.Called(config)
	return args.Error(0)
}

type clientMock struct {
	mock.Mock
}