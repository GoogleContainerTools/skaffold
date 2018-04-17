CREATE TABLE hexagons
(
    image varchar(500) NOT NULL,
    category varchar(500) NOT NULL,
    name varchar(500) NOT NULL,
    description varchar(500) NOT NULL,
    url varchar(500) NOT NULL
);

INSERT INTO hexagons(image, category, name, description, url) VALUES ('compute-engine.svg', 'Compute', 'Compute Engine', 'Virtual Machines, Disks, Network', 'https://cloud.google.com/compute/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('app-engine.svg', 'Compute', 'App Engine', 'Managed App Platform', 'https://cloud.google.com/appengine/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('kubernetes-engine.svg', 'Compute', 'Kubernetes Engine', 'Managed Kubernetes/Containers', 'https://cloud.google.com/kubernetes-engine/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-functions.svg', 'Compute', 'Cloud Functions', 'Serverless Microservices', 'https://cloud.google.com/functions/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('bigquery.svg', 'Big Data', 'BigQuery', 'Managed Data Warehouse/Analytics', 'https://cloud.google.com/bigquery/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-dataflow.svg', 'Big Data', 'Cloud Dataflow', 'Managed Data Processing', 'https://cloud.google.com/dataflow/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-dataproc.svg', 'Big Data', 'Cloud Dataproc', 'Managed Spark and Hadoop', 'https://cloud.google.com/dataproc/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-datalab.svg', 'Big Data', 'Cloud Datalab', 'Visualize and Explore Data', 'https://cloud.google.com/datalab/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-dataprep.svg', 'Big Data', 'Cloud Dataprep', 'Visual Data Preparation Tool', 'https://cloud.google.com/dataprep/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-pubsub.svg', 'Big Data', 'Cloud Pub/Sub', 'Distributed Real-time messaging', 'https://cloud.google.com/pubsub/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('genomics.svg', 'Big Data', 'Genomics', 'Managed Genomics Platform', 'https://cloud.google.com/genomics/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('data-studio.svg', 'Big Data', 'Data Studio', 'Collaborative Dashboards', 'https://cloud.google.com/data-studio/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-storage.svg', 'Storage and Databases', 'Cloud Storage', 'Object/File Storage & Serving', 'https://cloud.google.com/storage/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-sql.svg', 'Storage and Databases', 'Cloud SQL', 'Managed MySQL and PostgreSQL', 'https://cloud.google.com/sql/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-bigtable.svg', 'Storage and Databases', 'Cloud Bigtable', 'HBase compatible NoSQL', 'https://cloud.google.com/bigtable/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-spanner.svg', 'Storage and Databases', 'Cloud Spanner', 'Horizontally Scalable Relational DB', 'https://cloud.google.com/spanner/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-datastore.svg', 'Storage and Databases', 'Cloud Datastore', 'Horizontally Scalable Document DB', 'https://cloud.google.com/datastore/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('persistent-disk.svg', 'Storage and Databases', 'Persistent Disk', 'VM-attached disks', 'https://cloud.google.com/persistent-disk/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-machine-learning-engine.svg', 'Cloud AI', 'Cloud Machine Learning Engine', 'Managed TensorFlow/ML', 'https://cloud.google.com/ml-engine/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-jobs-api.svg', 'Cloud AI', 'Cloud Jobs Discovery', 'ML Job Search and Discovery', 'https://cloud.google.com/job-discovery/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-natural-language.svg', 'Cloud AI', 'Cloud Natural Language', 'Text Parsing and Analysis', 'https://cloud.google.com/natural-language/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-speech-api.svg', 'Cloud AI', 'Cloud Speech-to-Text', 'Convert Speech to Text', 'https://cloud.google.com/speech-to-text/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-translation-api.svg', 'Cloud AI', 'Cloud Translation API', 'Language Detection and Translation', 'https://cloud.google.com/translate/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-vision-api.svg', 'Cloud AI', 'Cloud Vision API', 'Image Recognition and Classification', 'https://cloud.google.com/vision/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-video-intelligence-api.svg', 'Cloud AI', 'Cloud Video Intelligence', 'Scene-level Video Annotation', 'https://cloud.google.com/video-intelligence/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-virtual-network.svg', 'Networking', 'Cirtual Private Cloud (VPC)', 'Software Defined Networking', 'https://cloud.google.com/vpc/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-load-balancing.svg', 'Networking', 'Cloud Load Balancing', 'Multi-region Load Distribution', 'https://cloud.google.com/load-balancing/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-cdn.svg', 'Networking', 'Cloud CDN', 'Content Delivery Network', 'https://cloud.google.com/cdn/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-interconnect.svg', 'Networking', 'Cloud Interconnect', 'Peer with GCP', 'https://cloud.google.com/interconnect/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-dns.svg', 'Networking', 'Cloud DNS', 'Programmable DNS Serving', 'https://cloud.google.com/dns/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-iam.svg', 'Identity & Security', 'Cloud IAM', 'Resource Access Control', 'https://cloud.google.com/iam/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-identity-aware-proxy.svg', 'Identity & Security', 'Cloud Identity-Aware Proxy', 'Identity-based App Signin', 'https://cloud.google.com/iap/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-data-loss-prevention-api.svg', 'Identity & Security', 'Cloud Data Loss Prevention API', 'Redact Sensitive Data', 'https://cloud.google.com/dlp/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('security-key-enforcement.svg', 'Identity & Security', 'Security Key Enforcement', '2-Step Key Verification', 'https://cloud.google.com/security-key/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-key-management-service.svg', 'Identity & Security', 'Cloud Key Management Service', 'Hosted Key Management Service', 'https://cloud.google.com/kms/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-resource-manager.svg', 'Identity & Security', 'Cloud Resource Manager', 'Cloud Project Metadata Management', 'https://cloud.google.com/resource-manager/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-security-scanner.svg', 'Identity & Security', 'Cloud Security Scanner', 'App Engine Security Scanner', 'https://cloud.google.com/security-scanner/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('monitoring.svg', 'Management Tools', 'Monitoring', 'Infrastructure and Application Monitoring', 'https://cloud.google.com/monitoring/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('logging.svg', 'Management Tools', 'Logging', 'Centralized Logging', 'https://cloud.google.com/logging/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('error-reporting.svg', 'Management Tools', 'Error Reporting', 'App Error Reporting', 'https://cloud.google.com/error-reporting/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('trace.svg', 'Management Tools', 'Trace', 'App Performance Insights', 'https://cloud.google.com/trace/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('debugger.svg', 'Management Tools', 'Debugger', 'App Debugging', 'https://cloud.google.com/debugger/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-deployment-manager.svg', 'Management Tools', 'Cloud Deployment Manager', 'Templated Infrastructure Deployment', 'https://cloud.google.com/deployment-manager/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-endpoints.svg', 'Management Tools', 'Cloud Endpoints', 'Cloud API Gateway', 'https://cloud.google.com/endpoints/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-console.svg', 'Management Tools', 'Cloud Console', 'Web-based Management Console', 'https://cloud.google.com/cloud-console/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-shell.svg', 'Management Tools', 'Cloud Shell', 'Browser-based Terminal/CLI', 'https://cloud.google.com/shell/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-mobile-app.svg', 'Management Tools', 'Cloud Mobile App', 'iOS/Android GCP Manager App', 'https://cloud.google.com/console-app/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-billing-api.svg', 'Management Tools', 'Cloud Billing API', 'Programmatically Manage GCP Billing', 'https://cloud.google.com/billing/docs/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-apis.svg', 'Management Tools', 'Cloud APIs', 'APIs for Cloud Services', 'https://cloud.google.com/apis/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-sdk.svg', 'Developer Tools', 'Cloud SDK', 'CLI for GCP', 'https://cloud.google.com/sdk/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('container-registry.svg', 'Developer Tools', 'Container Registry', 'Private Container Registry/Storage', 'https://cloud.google.com/container-registry/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('container-builder.svg', 'Developer Tools', 'Container Builder', 'Build/Package Container Artifacts', 'https://cloud.google.com/container-builder/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-source-repositories.svg', 'Developer Tools', 'Cloud Source Repositories', 'Hosted Private Git Repos', 'https://cloud.google.com/source-repositories/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-tools-for-android-studio.svg', 'Developer Tools', 'Cloud Tools for Android Studio', 'Android Studio GCP Tools', 'https://cloud.google.com/tools/android-studio/docs/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-tools-for-intellij.svg', 'Developer Tools', 'Cloud Tools for IntelliJ', 'IntelliJ GCP Tools', 'https://cloud.google.com/intellij/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-tools-for-powershell.svg', 'Developer Tools', 'Cloud Tools for PowerShell', 'PowerShell GCP Tools', 'https://cloud.google.com/powershell/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-tools-for-visual-studio.svg', 'Developer Tools', 'Cloud Tools for Visual Studio', 'Visual Studio GCP Tools', 'https://cloud.google.com/visual-studio/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-tools-for-eclipse.svg', 'Developer Tools', 'Cloud Tools for Eclipse', 'Eclipse GCP Tools', 'https://cloud.google.com/eclipse/docs/');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('gradle-app-engine-plugin.svg', 'Developer Tools', 'Gradle App Engine Plugin', 'Gradle App Engine Plugin', 'https://github.com/GoogleCloudPlatform/app-gradle-plugin');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('maven-app-engine-plugin.svg', 'Developer Tools', 'Maven App Engine Plugin', 'Maven App Engine Plugin', 'https://github.com/GoogleCloudPlatform/app-maven-plugin');
INSERT INTO hexagons(image, category, name, description, url) VALUES ('cloud-test-lab.svg', 'Developer Tools', 'Cloud Test Lab', 'Mobile Device Testing Service', 'https://firebase.google.com/docs/test-lab/');
