package vars

import "fmt"

const (
	dbURL = "%s@tcp(boulder-mysql:3306)/%s"
)

var (
	// DBConnSA is the sa database connection
	DBConnSA = fmt.Sprintf(dbURL, "sa", "boulder_sa_test")
	// DBConnSAMailer is the sa mailer database connection
	DBConnSAMailer = fmt.Sprintf(dbURL, "mailer", "boulder_sa_test")
	// DBConnSAFullPerms is the sa database connection with full perms
	DBConnSAFullPerms = fmt.Sprintf(dbURL, "test_setup", "boulder_sa_test")
	// DBConnSAOcspUpdateRO is the sa ocsp_update_ro database connection
	DBConnSAOcspUpdateRO = fmt.Sprintf(dbURL, "ocsp_update_ro", "boulder_sa_test")
	// DBInfoSchemaRoot is the root user and the information_schema connection.
	DBInfoSchemaRoot = fmt.Sprintf(dbURL, "root", "information_schema")
	// DBConnIncidents is the incidents database connection.
	DBConnIncidents = fmt.Sprintf(dbURL, "incidents_sa", "incidents_sa_test")
	// DBConnIncidentsFullPerms is the incidents database connection with full perms.
	DBConnIncidentsFullPerms = fmt.Sprintf(dbURL, "test_setup", "incidents_sa_test")
)
