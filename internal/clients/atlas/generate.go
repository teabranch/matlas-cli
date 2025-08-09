//go:generate go run go.uber.org/mock/mockgen -source=client.go -destination=mocks/mock_client.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=projects.go -destination=mocks/mock_projects.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=organizations.go -destination=mocks/mock_organizations.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=clusters.go -destination=mocks/mock_clusters.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=users.go -destination=mocks/mock_users.go -package=mocks
//go:generate go run go.uber.org/mock/mockgen -source=network_access.go -destination=mocks/mock_network_access.go -package=mocks

package atlas
