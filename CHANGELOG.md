# v0.22.0 Release - 1/31/2019

Note: This release comes with a config change, use `skaffold fix` to permanently upgrade your config to `v1beta4`, however old versions are now auto-upgraded. 
See [deprecation-policy.md](/deprecation-policy.md) for details on what beta means.

New features: 
* Introduce configuration option to configure image pushing per kube-context [#1355](https://github.com/GoogleContainerTools/skaffold/pull/1355)
* Better support for docker build with a target [#1497](https://github.com/GoogleContainerTools/skaffold/pull/1497)
* Reintroduce the fsNotify trigger [#1562](https://github.com/GoogleContainerTools/skaffold/pull/1562)
* Add zsh completion [#1531](https://github.com/GoogleContainerTools/skaffold/pull/1531)
* `#296` Support remote helm chart repositories [#1254](https://github.com/GoogleContainerTools/skaffold/pull/1254)

Fixes: 
* Fix bug in port forwarding [#1529](https://github.com/GoogleContainerTools/skaffold/pull/1529)
* Fix doc for Kustomize deploy: path option [#1527](https://github.com/GoogleContainerTools/skaffold/pull/1527)
* Fix broken links in Getting Started [#1523](https://github.com/GoogleContainerTools/skaffold/pull/1523)
* Use configured namespace for pod watcher. [#1473](https://github.com/GoogleContainerTools/skaffold/pull/1473)
* Pass DOCKER* env variables for jib to connect to minikube [#1505](https://github.com/GoogleContainerTools/skaffold/pull/1505)

Updates & Refactorings: 
* Upgrade to jib 1.0.0 [#1512](https://github.com/GoogleContainerTools/skaffold/pull/1512)
* Don’t use local Docker to push Bazel images [#1493](https://github.com/GoogleContainerTools/skaffold/pull/1493)
* Use kubectl to read the manifests [#1451](https://github.com/GoogleContainerTools/skaffold/pull/1451)
* Simplify integration tests [#1539](https://github.com/GoogleContainerTools/skaffold/pull/1539)
* Fix master branch [#1569](https://github.com/GoogleContainerTools/skaffold/pull/1569)
* Add missing tests for watch/triggers [#1557](https://github.com/GoogleContainerTools/skaffold/pull/1557)
* Improve triggers [#1561](https://github.com/GoogleContainerTools/skaffold/pull/1561)
* Add tests for labels package [#1534](https://github.com/GoogleContainerTools/skaffold/pull/1534)

Docs updates: 
* Fix skaffold.dev indexing on Google [#1547](https://github.com/GoogleContainerTools/skaffold/pull/1547)
* 2019 roadmap [#1530](https://github.com/GoogleContainerTools/skaffold/pull/1530)
* Should be v1beta3 [#1515](https://github.com/GoogleContainerTools/skaffold/pull/1515)
* Renaming the CoC for GitHub [#1518](https://github.com/GoogleContainerTools/skaffold/pull/1518)
* Add Priya as a Codeowner [#1544](https://github.com/GoogleContainerTools/skaffold/pull/1544)
* Add Priya as a maintainer [#1542](https://github.com/GoogleContainerTools/skaffold/pull/1542)
* Note JVM flags specific to Java 8 in examples/jib [#1563](https://github.com/GoogleContainerTools/skaffold/pull/1563)

Huge thanks goes out to all of our contributors for this release: 

- Balint Pato
- Brian de Alwis
- Cornelius Weig
- David Gageot
- Koen De Keyser
- Labesse Kévin
- Michael FIG
- Nick Kubala
- Priya Wadhwa
- Shuhei Kitagawa
- czhc


# v0.21.1 Release - 1/22/2019

New Features:
* Add a log when bazel deps take a long time [#1498](https://github.com/GoogleContainerTools/skaffold/pull/1498)
* Pre-pull cache-from images [#1495](https://github.com/GoogleContainerTools/skaffold/pull/1495)
* Pass bazel args to `bazel info bazel-bin` [#1487](https://github.com/GoogleContainerTools/skaffold/pull/1487)
* Support secretGenerators with kustomize  [#1488](https://github.com/GoogleContainerTools/skaffold/pull/1488)


Fixes:
* Fix coloured output when building in // [#1501](https://github.com/GoogleContainerTools/skaffold/pull/1501)
* Fix onbuild analysis [#1491](https://github.com/GoogleContainerTools/skaffold/pull/1491)
* Fix Broken link to references/config in documentation [#1486](https://github.com/GoogleContainerTools/skaffold/pull/1486)


Updates & refactorings:
* Add error for non Docker artifacts built with Kaniko [#1494](https://github.com/GoogleContainerTools/skaffold/pull/1494)
* Update bazel example [#1492](https://github.com/GoogleContainerTools/skaffold/pull/1492)
* Revert "Merge pull request #1439 from ltouati/fsnotify" [#1508](https://github.com/GoogleContainerTools/skaffold/pull/1508)
* Don’t log if nothing is copied or deleted [#1504](https://github.com/GoogleContainerTools/skaffold/pull/1504)
* Add more integration tests [#1502](https://github.com/GoogleContainerTools/skaffold/pull/1502)
* Remove file committed by error [#1500](https://github.com/GoogleContainerTools/skaffold/pull/1500)


Docs updates:
* Update doc around local development [#1446](https://github.com/GoogleContainerTools/skaffold/pull/1446)
* [doc] Fix default value for manifests [#1485](https://github.com/GoogleContainerTools/skaffold/pull/1485)

Huge thanks goes out to all of our contributors for this release: 

- David Gageot
- Nick Kubala
- Priya Wadhwa
- Shane Lee


# v0.21.0 Release - 1/17/2019

Note: This release comes with a config change, use `skaffold fix` to permanently upgrade your config to `v1beta2`, however old versions are now auto-upgraded. 
See [deprecation-policy.md](/deprecation-policy.md) for details on what beta means.

New Features:
* Add support for urls in deploy.kubectl.manifests [#1408](https://github.com/GoogleContainerTools/skaffold/pull/1408)
* Add some tests for Sync [#1406](https://github.com/GoogleContainerTools/skaffold/pull/1406)
* Get digest on push and imageID on build [#1428](https://github.com/GoogleContainerTools/skaffold/pull/1428)
* Implement a notification based watcher [#1439](https://github.com/GoogleContainerTools/skaffold/pull/1439)
* Add k8s version check to kustomize deployer [#1449](https://github.com/GoogleContainerTools/skaffold/pull/1449)
* Support new K8s context name in Docker Desktop [#1448](https://github.com/GoogleContainerTools/skaffold/pull/1448)
* Upload sources for any kind of artifact [#1477](https://github.com/GoogleContainerTools/skaffold/pull/1477)
* feat(docker creds) can mount docker config into kaniko pod [#1466](https://github.com/GoogleContainerTools/skaffold/pull/1466)
* Support Jib on Google Cloud Build [#1478](https://github.com/GoogleContainerTools/skaffold/pull/1478)


Fixes:
* fix search URL for skaffold.dev + github edit link [#1417](https://github.com/GoogleContainerTools/skaffold/pull/1417)
* Print error messages when containers can’t be started [#1415](https://github.com/GoogleContainerTools/skaffold/pull/1415)
* Script should be executable [#1423](https://github.com/GoogleContainerTools/skaffold/pull/1423)
* Fix port-forwarding not being triggered. [#1433](https://github.com/GoogleContainerTools/skaffold/pull/1433)
* Fix localDir context for Kaniko on Windows [#1438](https://github.com/GoogleContainerTools/skaffold/pull/1438)
* Remove spurious warning [#1442](https://github.com/GoogleContainerTools/skaffold/pull/1442)
* Test what was actually deployed [#1462](https://github.com/GoogleContainerTools/skaffold/pull/1462)
* Fix jib tagging [#1475](https://github.com/GoogleContainerTools/skaffold/pull/1475)


Updates & refactorings:
* Move trigger related code to the watcher [#1422](https://github.com/GoogleContainerTools/skaffold/pull/1422)
* Simplify fake docker api [#1424](https://github.com/GoogleContainerTools/skaffold/pull/1424)
* Small improvements gcb [#1425](https://github.com/GoogleContainerTools/skaffold/pull/1425)
* Small improvements to kaniko builder [#1426](https://github.com/GoogleContainerTools/skaffold/pull/1426)
* Update golangci lint [#1430](https://github.com/GoogleContainerTools/skaffold/pull/1430)
* Refactor docker api [#1429](https://github.com/GoogleContainerTools/skaffold/pull/1429)
* Use latest release of Jib [#1440](https://github.com/GoogleContainerTools/skaffold/pull/1440)
* Refactor FakeCmd [#1456](https://github.com/GoogleContainerTools/skaffold/pull/1456)
* Use cmd.Run() indirection [#1457](https://github.com/GoogleContainerTools/skaffold/pull/1457)
* Clear error message for unsupported artifact on GCB [#1453](https://github.com/GoogleContainerTools/skaffold/pull/1453)
* Improve port-forwarding [#1452](https://github.com/GoogleContainerTools/skaffold/pull/1452)
* Minor changes to kaniko builder [#1461](https://github.com/GoogleContainerTools/skaffold/pull/1461)
* Show duplication in jib code [#1454](https://github.com/GoogleContainerTools/skaffold/pull/1454)
* Remove some duplication in Jib builder [#1465](https://github.com/GoogleContainerTools/skaffold/pull/1465)
* Use Maven wrapper for Jib example easier start. [#1471](https://github.com/GoogleContainerTools/skaffold/pull/1471)
* Simplify docker.AddTag() [#1464](https://github.com/GoogleContainerTools/skaffold/pull/1464)
* Embed labelling into Deployers [#1463](https://github.com/GoogleContainerTools/skaffold/pull/1463)
* Refactor port forwarding [#1474](https://github.com/GoogleContainerTools/skaffold/pull/1474)


Docs updates:
* CLI reference docs automation [#1418](https://github.com/GoogleContainerTools/skaffold/pull/1418)
* installation link to readme [#1437](https://github.com/GoogleContainerTools/skaffold/pull/1437)
* docs: typo + add setValueTemplates usecase [#1450](https://github.com/GoogleContainerTools/
* fix(docs) updated references for imageName to be image [#1468](https://github.com/GoogleContainerTools/skaffold/pull/1468)
* More fixes to the builders doc [#1469](https://github.com/GoogleContainerTools/skaffold/pull/1469)
* fix: correct spelling of Kaninko to Kaniko [#1472](https://github.com/GoogleContainerTools/skaffold/pull/1472)

Huge thank you for this release towards our contributors:

- Balint Pato
- Bruno Miguel Custodio
- Cedric Kring
- David Gageot
- Gareth Evans
- George Oakling
- Ivan Portyankin
- Lionel Touati
- Matt Rickard
- Matti Paksula
- Nick Kubala
- Priya Wadhwa


# v0.20.0 Release - 12/21/2018

Note: This release comes with a config change, use `skaffold fix` to permanently upgrade your config to `v1beta2`, however old versions are now auto-upgraded. 
See [deprecation-policy.md](/deprecation-policy.md) for details on what beta means.

New Features: 

* Add additional flags to kaniko builder [#1387](https://github.com/GoogleContainerTools/skaffold/pull/1387)

Fixes: 

* Omit empty strings in jib sections of the config [#1399](https://github.com/GoogleContainerTools/skaffold/pull/1399)
* Don’t panic if image field is not of type string [#1386](https://github.com/GoogleContainerTools/skaffold/pull/1386)
* Fix Windows to Linux file sync by always converting path separators to *nix style [#1351](https://github.com/GoogleContainerTools/skaffold/pull/1351)
* Support labeling with hardcoded namespace [#1359](https://github.com/GoogleContainerTools/skaffold/pull/1359)
* Image name are case sensitive [#1342](https://github.com/GoogleContainerTools/skaffold/pull/1342)
* Print logs for containers that are not ready [#1344](https://github.com/GoogleContainerTools/skaffold/pull/1344)
* Cleanup only if something was actually deployed [#1343](https://github.com/GoogleContainerTools/skaffold/pull/1343)
* Don’t assume bazel-bin is symlinked in workspace [#1340](https://github.com/GoogleContainerTools/skaffold/pull/1340)


Updates & refactorings: 

* Cleanup tagger tests [#1375](https://github.com/GoogleContainerTools/skaffold/pull/1375)
* Local builders return a digest [#1374](https://github.com/GoogleContainerTools/skaffold/pull/1374)
* Remove kqueue tag [#1402](https://github.com/GoogleContainerTools/skaffold/pull/1402)
* Improve runner unit tests [#1398](https://github.com/GoogleContainerTools/skaffold/pull/1398)
* Create directory before kubectl cp [#1390](https://github.com/GoogleContainerTools/skaffold/pull/1390)
* Add missing fake k8s context [#1384](https://github.com/GoogleContainerTools/skaffold/pull/1384)
* Improve schema upgrade tests [#1383](https://github.com/GoogleContainerTools/skaffold/pull/1383)
* Update kaniko image to latest version [#1381](https://github.com/GoogleContainerTools/skaffold/pull/1381)
* Introduce config version v1beta2 [#1376](https://github.com/GoogleContainerTools/skaffold/pull/1376)
* Tag image by digest [#1367](https://github.com/GoogleContainerTools/skaffold/pull/1367)
* Pass tag options by value [#1372](https://github.com/GoogleContainerTools/skaffold/pull/1372)
* Extract push/no-push logic into builder [#1366](https://github.com/GoogleContainerTools/skaffold/pull/1366)
* keeping integration test only examples under integration tests [#1362](https://github.com/GoogleContainerTools/skaffold/pull/1362)
* Display usage tips to the user [#1361](https://github.com/GoogleContainerTools/skaffold/pull/1361)
* Handle errors in release walking [#1356](https://github.com/GoogleContainerTools/skaffold/pull/1356)

Docs updates: 

* new skaffold site [#1338](https://github.com/GoogleContainerTools/skaffold/pull/1338)

Utilities: 

* If webhook deployment fails, upload logs [#1348](https://github.com/GoogleContainerTools/skaffold/pull/1348)

Huge thank you for this release towards our contributors: 

- Balint Pato
- David Gageot
- Gareth Evans
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- Travis Cline
- Valery Vitko


# v0.19.0 Release - 11/29/2018

Note: This release comes with a config change, use `skaffold fix` to permanently upgrade your config to `v1beta1`, however old versions are now auto-upgraded. 
See [deprecation-policy.md](/deprecation-policy.md) for details on what beta means.   

New features: 

* Run tests in skaffold build, add `skip-tests` flag to skip tests [#1326](https://github.com/GoogleContainerTools/skaffold/pull/1326)
* Allow ** glob pattern in sync parameters [#1266](https://github.com/GoogleContainerTools/skaffold/pull/1266)
* Add caching to kaniko builder [#1287](https://github.com/GoogleContainerTools/skaffold/pull/1287)
* Support slashes in file sync glob patterns on windows [#1280](https://github.com/GoogleContainerTools/skaffold/pull/1280)
* Add --compose-file option to skaffold init [#1282](https://github.com/GoogleContainerTools/skaffold/pull/1282)
* Automatically fix old configs by default [#1259](https://github.com/GoogleContainerTools/skaffold/pull/1259)
* adding skaffold version to the docker push user agent [#1260](https://github.com/GoogleContainerTools/skaffold/pull/1260)

Fixes: 

* Fix node security issue [#1323](https://github.com/GoogleContainerTools/skaffold/pull/1323)
* Allow passing arguments to bazel build [#1289](https://github.com/GoogleContainerTools/skaffold/pull/1289)
* Get tmp Directory from os env in kaniko local context storing [#1285](https://github.com/GoogleContainerTools/skaffold/pull/1285)


Updates & refactorings: 

* Apply default values upgraded configurations [#1332](https://github.com/GoogleContainerTools/skaffold/pull/1332)
* Remove duplication between run and deploy [#1331](https://github.com/GoogleContainerTools/skaffold/pull/1331)
* Remove pointer to runtime.Object interface [#1329](https://github.com/GoogleContainerTools/skaffold/pull/1329)
* Shorter logs [#1335](https://github.com/GoogleContainerTools/skaffold/pull/1335)
* Update deps [#1333](https://github.com/GoogleContainerTools/skaffold/pull/1333)
* dep ensure && dep prune [#1297](https://github.com/GoogleContainerTools/skaffold/pull/1297)
* Should support v1alpha5 [#1314](https://github.com/GoogleContainerTools/skaffold/pull/1314)
* Improve kubernetes.Logger [#1309](https://github.com/GoogleContainerTools/skaffold/pull/1309)
* introduce v1beta1 config [#1305](https://github.com/GoogleContainerTools/skaffold/pull/1305)
* Simpler Runner [#1304](https://github.com/GoogleContainerTools/skaffold/pull/1304)
* Don’t run tests if nothing was built [#1302](https://github.com/GoogleContainerTools/skaffold/pull/1302)
* Simplify the Runner's tests [#1303](https://github.com/GoogleContainerTools/skaffold/pull/1303)
* removing the artifacts from appveyor [#1300](https://github.com/GoogleContainerTools/skaffold/pull/1300)
* The multi-deployer feature is not working. Remove it [#1291](https://github.com/GoogleContainerTools/skaffold/pull/1291)

Breaking changes: 

* Remove ACR builder [#1308](https://github.com/GoogleContainerTools/skaffold/pull/1308)
* Remove `quiet` command line flag [#1292](https://github.com/GoogleContainerTools/skaffold/pull/1292)

Docs updates: 

* Clarify what manifest paths are relative to when specifying in skaffold yaml [#1336](https://github.com/GoogleContainerTools/skaffold/pull/1336)
* adding deprecation policy and document component stability [#1324](https://github.com/GoogleContainerTools/skaffold/pull/1324)
* Add missing fields to annotated-skaffold.yaml [#1310](https://github.com/GoogleContainerTools/skaffold/pull/1310)
* brew install skaffold [#1290](https://github.com/GoogleContainerTools/skaffold/pull/1290)
* Lists indented in the installation section (minor fix) [#1298](https://github.com/GoogleContainerTools/skaffold/pull/1298)
* Make usage messages look like the others. [#1267](https://github.com/GoogleContainerTools/skaffold/pull/1267)

Utilities: 

* [docs-webhook] remove docs-modifications label from issue instead of deleting the label [#1306](https://github.com/GoogleContainerTools/skaffold/pull/1306)
* [docs-webhook] hugo extended version + nodejs  [#1295](https://github.com/GoogleContainerTools/skaffold/pull/1295)
* [docs-webhook] Release latest version of docs controller image [#1293](https://github.com/GoogleContainerTools/skaffold/pull/1293)
* [docs-webhook] upgrading hugo + unpinning webhook image [#1288](https://github.com/GoogleContainerTools/skaffold/pull/1288)
* [lint] Golangci lint upgrade [#1281](https://github.com/GoogleContainerTools/skaffold/pull/1281)
* [compilation] Support system's LDFLAGS, make compilation reproducible [#1270](https://github.com/GoogleContainerTools/skaffold/pull/1270)

Huge thank you for this release towards our contributors: 
- Balint Pato
- Cedric Vidal
- David Gageot
- Igor Zibarev
- Ihor Dvoretskyi
- Jamie Lennox
- Maxim Baz
- Nick Kubala
- Pascal Ehlert
- Priya Wadhwa
- Venkatesh


# v0.18.0 Release - 11/08/2018

Bug Fixes: 

* Don't lose test configuration when running skaffold fix [#1251](https://github.com/GoogleContainerTools/skaffold/pull/1251)
* Fix jib errors on ctrl-c [#1248](https://github.com/GoogleContainerTools/skaffold/pull/1248)
* Fix sync [#1253](https://github.com/GoogleContainerTools/skaffold/pull/1253)
* Update examples and release notes to use v1alpha5 [#1244](https://github.com/GoogleContainerTools/skaffold/pull/1244)
* Set Kind on `skaffold init` [#1237](https://github.com/GoogleContainerTools/skaffold/pull/1237)
* Do not print the manifest on to stdout when doing a deploy by kustomize [#1234](https://github.com/GoogleContainerTools/skaffold/pull/1234)
* Fixed panic if skaffold.yaml is empty (#1216) [#1221](https://github.com/GoogleContainerTools/skaffold/pull/1221)
* Suppress fatal error reporting when ^C skaffold with jib [#1228](https://github.com/GoogleContainerTools/skaffold/pull/1228)
* portforward for resources with hardcoded namespace [#1223](https://github.com/GoogleContainerTools/skaffold/pull/1223)
 
Updates: 

* Output config version in skaffold version [#1252](https://github.com/GoogleContainerTools/skaffold/pull/1252)
* Port forward multiple ports [#1250](https://github.com/GoogleContainerTools/skaffold/pull/1250)
* Improve errors [#1255](https://github.com/GoogleContainerTools/skaffold/pull/1255)
* Move structure tests out of getting-started example [#1220](https://github.com/GoogleContainerTools/skaffold/pull/1220)
* changes related to our docs review flow: 
  * Add github pkg to webhook [#1230](https://github.com/GoogleContainerTools/skaffold/pull/1230)
  * Allow webhook to create a deployment [#1227](https://github.com/GoogleContainerTools/skaffold/pull/1227)
  * Add hugo and git to webhook image [#1226](https://github.com/GoogleContainerTools/skaffold/pull/1226)
  * Add support for creating a service from webhook [#1213](https://github.com/GoogleContainerTools/skaffold/pull/1213)

Huge thank you for this release towards our contributors: 
- Balint Pato
- Brian de Alwis
- David Gageot
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- RaviTezu
- varunkashyap

# v0.17.0 Release - 10/26/2018

Note: This release comes with a config change, use `skaffold fix` to upgrade your config to `v1alpha5`.
We 'skipped' `v1alpha4` due to an accidental merge: see [#1235](https://github.com/GoogleContainerTools/skaffold/issues/1235#issuecomment-436429009)

New Features: 

* Add support for setting default-repo in global config [#1057](https://github.com/GoogleContainerTools/skaffold/pull/1057)
* Add support for building Maven multimodule projects [#1152](https://github.com/GoogleContainerTools/skaffold/pull/1152)
* Azure Container Registry runner [#1107](https://github.com/GoogleContainerTools/skaffold/pull/1107)

Bug fixes: 

* Improve Kaniko builder [#1168](https://github.com/GoogleContainerTools/skaffold/pull/1168)
* Use os.SameFile() to check for mvnw working-dir echo bug [#1167](https://github.com/GoogleContainerTools/skaffold/pull/1167)
* Fix kaniko default behaviour [#1139](https://github.com/GoogleContainerTools/skaffold/pull/1139)

Updates: 

* Change SkaffoldOption Labeller to not include a comma in the label value [#1169](https://github.com/GoogleContainerTools/skaffold/pull/1169)
* Remove annoying log [#1163](https://github.com/GoogleContainerTools/skaffold/pull/1163)
* Prepare next version of the config file [#1146](https://github.com/GoogleContainerTools/skaffold/pull/1146)
* Improve error handling for `completion` command [#1206](https://github.com/GoogleContainerTools/skaffold/pull/1206)
* Jib sample [#1147](https://github.com/GoogleContainerTools/skaffold/pull/1147)
* Node.js example with dependency handling and hot-reload [#1148](https://github.com/GoogleContainerTools/skaffold/pull/1148)

Huge thank you for this release towards our contributors: 
- Balint Pato
- Brian de Alwis
- Cedric Kring
- David Gageot
- Geert-Johan Riemer
- Martino Fornasa
- Matt Rickard
- Nick Kubala
- Priya Wadhwa
- foo0x29a
- varunkashyap


# v0.16.0 Release - 10/11/2018

New Features:
* Add a `skaffold diagnose` command [#1109](https://github.com/GoogleContainerTools/skaffold/pull/1109)
* Add localdir buildcontext to kaniko builder [#983](https://github.com/GoogleContainerTools/skaffold/pull/983)
* Add --label flag to specify custom labels for deployments [#1098](https://github.com/GoogleContainerTools/skaffold/pull/1098)
* Add support for building projects using jib [#1073](https://github.com/GoogleContainerTools/skaffold/pull/1073)

Bug Fixes:
* Fix jib artifacts in skaffold diagnose [#1141](https://github.com/GoogleContainerTools/skaffold/pull/1141)
* Fix master [#1133](https://github.com/GoogleContainerTools/skaffold/pull/1133)
* Delete and redeploy object upon error 'field is immutable' [#940](https://github.com/GoogleContainerTools/skaffold/pull/940)
* Fix `skaffold fix` [#1123](https://github.com/GoogleContainerTools/skaffold/pull/1123)
* Lists files recursively in jib.getDependencies and other fixes. [#1097](https://github.com/GoogleContainerTools/skaffold/pull/1097)
* Merge error? [#1095](https://github.com/GoogleContainerTools/skaffold/pull/1095)
* Fix missing parenthesis [#1089](https://github.com/GoogleContainerTools/skaffold/pull/1089)

Updates:
* Move sync code to pkg/skaffold/sync/kubectl [#1138](https://github.com/GoogleContainerTools/skaffold/pull/1138)
* Add a test to check version upgrades [#1103](https://github.com/GoogleContainerTools/skaffold/pull/1103)
* Add a way to unset global config values [#1086](https://github.com/GoogleContainerTools/skaffold/pull/1086)
* Handles Jib build directly to registry when push=true. [#1132](https://github.com/GoogleContainerTools/skaffold/pull/1132)
* Simplify Jib code [#1130](https://github.com/GoogleContainerTools/skaffold/pull/1130)
* Trim the dockerfile a bit: [#1128](https://github.com/GoogleContainerTools/skaffold/pull/1128)
* Pass context when listing dependencies [#1108](https://github.com/GoogleContainerTools/skaffold/pull/1108)
* Remove fully qualified jib path for maven goals [#1129](https://github.com/GoogleContainerTools/skaffold/pull/1129)
* Merge master into jib_skaffold feature branch. [#1117](https://github.com/GoogleContainerTools/skaffold/pull/1117)
* Merge Jib feature-branch [#1063](https://github.com/GoogleContainerTools/skaffold/pull/1063)
* Improves jib.getDependencies. [#1125](https://github.com/GoogleContainerTools/skaffold/pull/1125)
* skipPush -> push [#1114](https://github.com/GoogleContainerTools/skaffold/pull/1114)
* Support for dot files in dockerignore [#1122](https://github.com/GoogleContainerTools/skaffold/pull/1122)
* remove project level skaffold.yaml [#1119](https://github.com/GoogleContainerTools/skaffold/pull/1119)
* Merge master into jib_skaffold feature branch [#1116](https://github.com/GoogleContainerTools/skaffold/pull/1116)
* Unify Jib command wrapper usage [#1105](https://github.com/GoogleContainerTools/skaffold/pull/1105)
* Update labels when deploying to namespace other than default [#1115](https://github.com/GoogleContainerTools/skaffold/pull/1115)
* Improve sync [#1102](https://github.com/GoogleContainerTools/skaffold/pull/1102)
* Rename SkaffoldConfig to SkaffoldPipeline [#1087](https://github.com/GoogleContainerTools/skaffold/pull/1087)
* Kaniko improvements [#1101](https://github.com/GoogleContainerTools/skaffold/pull/1101)
* File Sync for skaffold dev [#1039](https://github.com/GoogleContainerTools/skaffold/pull/1039)
* Implement a manual trigger for watch mode [#1085](https://github.com/GoogleContainerTools/skaffold/pull/1085)
* Skaffold init asks user to write skaffold.yaml [#1091](https://github.com/GoogleContainerTools/skaffold/pull/1091)
* Couple of improvements to the test phase [#1080](https://github.com/GoogleContainerTools/skaffold/pull/1080)
* Merges branch 'master' into jib_skaffold. [#1088](https://github.com/GoogleContainerTools/skaffold/pull/1088)
* Implements jib.GetDependenciesMaven/GetDependenciesGradle. [#1058](https://github.com/GoogleContainerTools/skaffold/pull/1058)
* Add test runner [#1013](https://github.com/GoogleContainerTools/skaffold/pull/1013)
* Simplify schema versioning [#1067](https://github.com/GoogleContainerTools/skaffold/pull/1067)
* Changelog changes for v0.15.1 [#1075](https://github.com/GoogleContainerTools/skaffold/pull/1075)
* Minor logging improvements [#1142](https://github.com/GoogleContainerTools/skaffold/pull/1142)


# v0.15.1 Release - 10/02/2018

This is a minor release to address an inconsistency in the `skaffold fix` upgrade:
 
* Transform values files in profiles to v1alpha3 [#1070](https://github.com/GoogleContainerTools/skaffold/pull/1070)


# v0.15.0 Release - 9/27/2018

New Features:
* Added kustomize to deploy types [#1027](https://github.com/GoogleContainerTools/skaffold/pull/1027)
* Basic support for watching Kustomize dependencies [#1015](https://github.com/GoogleContainerTools/skaffold/pull/1015)
* Basic support for using kubectl and helm together [#586](https://github.com/GoogleContainerTools/skaffold/pull/586)
* Add support for multiple helm values files [#985](https://github.com/GoogleContainerTools/skaffold/pull/985)
* Add v1alpha3 Config [#982](https://github.com/GoogleContainerTools/skaffold/pull/982)

Bug Fixes:
* annotated.yaml: fix gcb timeout format [#1040](https://github.com/GoogleContainerTools/skaffold/pull/1040)
* Catch a 409 when creating a bucket and continue. [#1044](https://github.com/GoogleContainerTools/skaffold/pull/1044)
* Fix typo [#1045](https://github.com/GoogleContainerTools/skaffold/pull/1045)
* Fix issues with build args replacement [#1028](https://github.com/GoogleContainerTools/skaffold/pull/1028)
* prevent watcher failure if helm valuesFilePath not set [#930](https://github.com/GoogleContainerTools/skaffold/pull/930)
* Correctly parse build tags that contain port numbers [#1001](https://github.com/GoogleContainerTools/skaffold/pull/1001)
* FIX kubectl should only redeploy updated manifests [#1014](https://github.com/GoogleContainerTools/skaffold/pull/1014)
* Fix race conditions in TestWatch [#987](https://github.com/GoogleContainerTools/skaffold/pull/987)

Updates:
* Simpler merged PR collection for release notes [#1054](https://github.com/GoogleContainerTools/skaffold/pull/1054)
* Improve kustomize deployer [#1036](https://github.com/GoogleContainerTools/skaffold/pull/1036)
* kustomizePath is a folder that defaults to . [#1030](https://github.com/GoogleContainerTools/skaffold/pull/1030)
* Discard output in tests [#1021](https://github.com/GoogleContainerTools/skaffold/pull/1021)
* Add a test for `kubectl should only redeploy updated manifests` [#1022](https://github.com/GoogleContainerTools/skaffold/pull/1022)
* Examples versioning [#1019](https://github.com/GoogleContainerTools/skaffold/pull/1019)
* add nkubala to MAINTAINERS [#993](https://github.com/GoogleContainerTools/skaffold/pull/993)
* Debounce rapid file changes [#1005](https://github.com/GoogleContainerTools/skaffold/pull/1005)
* Print kubectl client version [#991](https://github.com/GoogleContainerTools/skaffold/pull/991)
* Auto configure authentication helper for gcr.io [#989](https://github.com/GoogleContainerTools/skaffold/pull/989)
* Tweak the Dockerfile. [#1007](https://github.com/GoogleContainerTools/skaffold/pull/1007)
* Skip kaniko-related test when running locally [#990](https://github.com/GoogleContainerTools/skaffold/pull/990)
* Extract code from GCB [#986](https://github.com/GoogleContainerTools/skaffold/pull/986)


# v0.14.0 Release - 9/13/2018

New Features:
* Allow `skaffold dev —watch image` [#925](https://github.com/GoogleContainerTools/skaffold/pull/925)
* Port forward pods automatically during `skaffold dev` [#945](https://github.com/GoogleContainerTools/skaffold/pull/945)
* Add skaffold 'init' [#919](https://github.com/GoogleContainerTools/skaffold/pull/919)

Bug Fixes:
* Get namespace for updating objects from build artifact [#951](https://github.com/GoogleContainerTools/skaffold/pull/951)
* Remove service labeling temporarily [#965](https://github.com/GoogleContainerTools/skaffold/pull/965)
* Don't prefix pod names when port forwarding [#976](https://github.com/GoogleContainerTools/skaffold/pull/976)

Updates:
* Don’t compute onbuild triggers for images that are stage names [#938](https://github.com/GoogleContainerTools/skaffold/pull/938)
* Don't unmute logs if an error happened [#928](https://github.com/GoogleContainerTools/skaffold/pull/928)
* Exclude helm dependency chart packages from watched files [#932](https://github.com/GoogleContainerTools/skaffold/pull/932)
* Pass --recreate-pods to helm by default in dev mode [#946](https://github.com/GoogleContainerTools/skaffold/pull/946)
* Default to kubectl deploy [#956](https://github.com/GoogleContainerTools/skaffold/pull/956)
* Simplify helm tests [#957](https://github.com/GoogleContainerTools/skaffold/pull/957)
* Pull 'cache-from' images on Google Cloud Build [#958](https://github.com/GoogleContainerTools/skaffold/pull/958)
* update check respected quiet flag [#964](https://github.com/GoogleContainerTools/skaffold/pull/964)
* Fix typo in portforwarder [#975](https://github.com/GoogleContainerTools/skaffold/pull/975)


# v0.13.0 Release - 8/16/2018

New Features:
* Add --tail flag to stream logs with skaffold run [#914](https://github.com/GoogleContainerTools/skaffold/pull/914)
* Add DEVELOPMENT.md [#901](https://github.com/GoogleContainerTools/skaffold/pull/901)

Bug Fixes: 
* fixes `skaffold version` in the released docker image [#933](https://github.com/GoogleContainerTools/skaffold/pull/933)

Updates:
* as a base for future features - global skaffold config [#896](https://github.com/GoogleContainerTools/skaffold/pull/896)
* Remove duplication in kustomize deployer [#900](https://github.com/GoogleContainerTools/skaffold/pull/900)
* update readme with documentation links [#908](https://github.com/GoogleContainerTools/skaffold/pull/908)
* Fix a typo in "annotated-skaffold.yaml" [#907](https://github.com/GoogleContainerTools/skaffold/pull/907)
* Decouple visiting manifests and replacing images [#909](https://github.com/GoogleContainerTools/skaffold/pull/909)
* Add a simple test for Watcher [#898](https://github.com/GoogleContainerTools/skaffold/pull/898)
* Add test for signal handling [#917](https://github.com/GoogleContainerTools/skaffold/pull/917)
* Add the --target flag as a parameter to the docker builder. [#894](https://github.com/GoogleContainerTools/skaffold/pull/894)
* Misc improvements [#911](https://github.com/GoogleContainerTools/skaffold/pull/911)
* Add --tail flag to stream logs with skaffold run [#914](https://github.com/GoogleContainerTools/skaffold/pull/914)
* Extract code to tail logs [#924](https://github.com/GoogleContainerTools/skaffold/pull/924)
* Improve logs [#918](https://github.com/GoogleContainerTools/skaffold/pull/918)
* Add yamltags [#388](https://github.com/GoogleContainerTools/skaffold/pull/388)
* adding wrapper script for release note generation  [#935](https://github.com/GoogleContainerTools/skaffold/pull/935)
* detete -> delete [#941](https://github.com/GoogleContainerTools/skaffold/pull/941)


# v0.12.0 Release - 8/16/2018
New Features:
* Update check [#866](https://github.com/GoogleContainerTools/skaffold/pull/866)
* Simpler and faster git tagger [#846](https://github.com/GoogleContainerTools/skaffold/pull/846)
* Support setting namespace for every deployer [#852](https://github.com/GoogleContainerTools/skaffold/pull/852)
* Improve Cloud Build builder [#874](https://github.com/GoogleContainerTools/skaffold/pull/874)
* Improve file change tracking [#888](https://github.com/GoogleContainerTools/skaffold/pull/888)


Bug Fixes:
* Run Kaniko builds in parallel [#876](https://github.com/GoogleContainerTools/skaffold/pull/876)
* Do not run kubectl if nothing has changed [#877](https://github.com/GoogleContainerTools/skaffold/pull/877)
* fix version in released docker image [#878](https://github.com/GoogleContainerTools/skaffold/pull/878)
* Fix integration tests [#881](https://github.com/GoogleContainerTools/skaffold/pull/881)

Updates:
* Run Kaniko builds in parallel [#876](https://github.com/GoogleContainerTools/skaffold/pull/876)
* Watch mode 4th edition [#833](https://github.com/GoogleContainerTools/skaffold/pull/833)
* add bazel to skaffold docker image, add integration test for bazel [#879](https://github.com/GoogleContainerTools/skaffold/pull/879)
* Add missing filename to error message [#880](https://github.com/GoogleContainerTools/skaffold/pull/880)
* Fix minor lint errors surfaced by the 'misspell' and 'unparam' lint modules [#883](https://github.com/GoogleContainerTools/skaffold/pull/883)
* Update golangci-lint to v1.9.3 and enable misspell+unparam modules [#884](https://github.com/GoogleContainerTools/skaffold/pull/884)
* add codecov to travis and repo [#885](https://github.com/GoogleContainerTools/skaffold/pull/885)
* Add test helper to handle actions on tmp dirs [#893](https://github.com/GoogleContainerTools/skaffold/pull/893)
* Use reflection to overlay profile onto config [#872](https://github.com/GoogleContainerTools/skaffold/pull/872)


# v0.11.0 Release - 8/02/2018
New Features: 
* Pass buildArgs to Kaniko [#822](https://github.com/GoogleContainerTools/skaffold/pull/822)
* Add pop of color to terminal output with a color formatter [#857](https://github.com/GoogleContainerTools/skaffold/pull/857)

Bug Fixes:
* Substitute build args from config into parsed Dockerfile before processing deps [#828](https://github.com/GoogleContainerTools/skaffold/pull/828)
* Fix color.Fprintln bug [#861](https://github.com/GoogleContainerTools/skaffold/pull/861)
* Issue #836: Use releaseName to get release info. [#855](https://github.com/GoogleContainerTools/skaffold/pull/855)
* Switch to gcr for the kaniko builder example. [#845](https://github.com/GoogleContainerTools/skaffold/pull/845)

Updates:
* boilerplate.sh: fail if python script not found; run from any dir [#827](https://github.com/GoogleContainerTools/skaffold/pull/827)
* Revert to default grace period [#815](https://github.com/GoogleContainerTools/skaffold/pull/815)
* Skip the deployment if no manifests are defined [#832](https://github.com/GoogleContainerTools/skaffold/pull/832)
* Slightly faster git tagger [#839](https://github.com/GoogleContainerTools/skaffold/pull/839)
* Don’t tag the same images twice [#842](https://github.com/GoogleContainerTools/skaffold/pull/842)
* Faster code to get image digest [#838](https://github.com/GoogleContainerTools/skaffold/pull/838)
* Simpler code to print Kaniko logs [#831](https://github.com/GoogleContainerTools/skaffold/pull/831)
* Simpler sha256 tagger code [#847](https://github.com/GoogleContainerTools/skaffold/pull/847)
* Move builders to sub packages [#830](https://github.com/GoogleContainerTools/skaffold/pull/830)
* Shell out docker build [#840](https://github.com/GoogleContainerTools/skaffold/pull/840)
* Don’t redeploy twice the same manifest in a dev loop [#843](https://github.com/GoogleContainerTools/skaffold/pull/843)
* Remove `skaffold docker` commands [#853](https://github.com/GoogleContainerTools/skaffold/pull/853)
* Find docker deps 10x faster [#837](https://github.com/GoogleContainerTools/skaffold/pull/837)
* Simplify docker related code. [#854](https://github.com/GoogleContainerTools/skaffold/pull/854)
* add support for helm image convention vs fqn setting [#826](https://github.com/GoogleContainerTools/skaffold/pull/826)
* Update dep to v0.5.0 [#862](https://github.com/GoogleContainerTools/skaffold/pull/862)


# v0.10.0 Release - 7/13/2018
New Features:
* kustomize: use custom path in deploy deps [#766](https://github.com/GoogleContainerTools/skaffold/pull/766)
* helm: add deploy dependency paths [#765](https://github.com/GoogleContainerTools/skaffold/pull/765)
* Use digest when the git repo has no commit [#794](https://github.com/GoogleContainerTools/skaffold/pull/794)
* GCB now builds artifacts in // [#805](https://github.com/GoogleContainerTools/skaffold/pull/805)
* Default kubectl manifests to `k8s/*.yaml` [#810](https://github.com/GoogleContainerTools/skaffold/pull/810)
* Support disk size and machine type for GCB [#808](https://github.com/GoogleContainerTools/skaffold/pull/808)
* Support additional flags for kubectl commands [#807](https://github.com/GoogleContainerTools/skaffold/pull/807)
* Try to guess GCB projectID from the image name [#809](https://github.com/GoogleContainerTools/skaffold/pull/809)

Bug Fixes:
* kustomize: cleanup custom kustomize path [#781](https://github.com/GoogleContainerTools/skaffold/pull/781)
* corrected region typo [#792](https://github.com/GoogleContainerTools/skaffold/pull/792)
* Fixed a small typo in docs [#797](https://github.com/GoogleContainerTools/skaffold/pull/797)
* Small code changes [#796](https://github.com/GoogleContainerTools/skaffold/pull/796)

Updates:
* docs: alphabetize readme peoples [#764](https://github.com/GoogleContainerTools/skaffold/pull/764)
* makefile: redirection for checksums [#768](https://github.com/GoogleContainerTools/skaffold/pull/768)
* brew: remove version from formula [#763](https://github.com/GoogleContainerTools/skaffold/pull/763)
* Add the logo [#774](https://github.com/GoogleContainerTools/skaffold/pull/774)
* ci: also push latest skaffold image on commit [#773](https://github.com/GoogleContainerTools/skaffold/pull/773)
* tests: pin golangci-lint version to v1.8.1 [#780](https://github.com/GoogleContainerTools/skaffold/pull/780)
* Remove dead code [#784](https://github.com/GoogleContainerTools/skaffold/pull/784)
* Improve GCR docs [#795](https://github.com/GoogleContainerTools/skaffold/pull/795)
* Extract code to build a single artifact locally [#798](https://github.com/GoogleContainerTools/skaffold/pull/798)
* Use dynamic client for labels [#782](https://github.com/GoogleContainerTools/skaffold/pull/782)
* Update Kaniko to v0.2.0 [#803](https://github.com/GoogleContainerTools/skaffold/pull/803)
* Upgrade k8s dependency to 1.11.0 [#804](https://github.com/GoogleContainerTools/skaffold/pull/804)
* Fix missing logs [#786](https://github.com/GoogleContainerTools/skaffold/pull/786)
* calculate version from git [#814](https://github.com/GoogleContainerTools/skaffold/pull/814)
* logs: use namespace flag when streaming pods [#819](https://github.com/GoogleContainerTools/skaffold/pull/819)

# v0.9.0 Release - 6/28/2018
New Features:
* Print the image name that's being built [#732](https://github.com/GoogleContainerTools/skaffold/pull/732)
* Publish windows binaries on AppVeyor [#738](https://github.com/GoogleContainerTools/skaffold/pull/738)
* Add labeling for profiles [#736](https://github.com/GoogleContainerTools/skaffold/pull/736)
* Improve Git tagger [#714](https://github.com/GoogleContainerTools/skaffold/pull/714)
* Support docker build --cache-from [#737](https://github.com/GoogleContainerTools/skaffold/pull/737)
* Add custom kustomization path [#749](https://github.com/GoogleContainerTools/skaffold/pull/749)
* Use tags only in case of perfect match [#755](https://github.com/GoogleContainerTools/skaffold/pull/755)

Bug Fixes:
* fixed a bug in dirtyTag which may leave extra whitespaces in changedPath [#721](https://github.com/GoogleContainerTools/skaffold/pull/721)
* Remove duplication in code handling labels [#723](https://github.com/GoogleContainerTools/skaffold/pull/723)
* Fix: Links for D4M Edge and D4W Edge were swapped [#735](https://github.com/GoogleContainerTools/skaffold/pull/735)
* Fix bug where dirty submodules broke hash generation [#711](https://github.com/GoogleContainerTools/skaffold/pull/711)
* Remove warning for an image that’s built and used by fqn [#713](https://github.com/GoogleContainerTools/skaffold/pull/713)
* Don’t always fail if some COPY patterns don't match any file [#744](https://github.com/GoogleContainerTools/skaffold/pull/744)
* Fix dev loop [#758](https://github.com/GoogleContainerTools/skaffold/pull/758)
* Fix kaniko defaults [#756](https://github.com/GoogleContainerTools/skaffold/pull/756)
* Don’t complain when object is not found during cleanup [#759](https://github.com/GoogleContainerTools/skaffold/pull/759)

Updates:
* Deployers should only rely on their specific config [#739](https://github.com/GoogleContainerTools/skaffold/pull/739)
* Builders should only rely on their specific config [#740](https://github.com/GoogleContainerTools/skaffold/pull/740)
* e2e test for helm deployments. [#743](https://github.com/GoogleContainerTools/skaffold/pull/743)
* New code to watch file changes [#620](https://github.com/GoogleContainerTools/skaffold/pull/620)
* docs: add info about published artifacts [#751](https://github.com/GoogleContainerTools/skaffold/pull/751)

# v0.8.0 Release - 06/21/2018

New Features
* cloudbuild: publish skaffold images on commit and tag [#655](https://github.com/GoogleContainerTools/skaffold/pull/655)
* Asciidocs and refdocs tooling [#648](https://github.com/GoogleContainerTools/skaffold/pull/648)
* Add support for skaffold.yml as a default config file fixes #225 [#665](https://github.com/GoogleContainerTools/skaffold/pull/665)
* adds helper script for release notes [#662](https://github.com/GoogleContainerTools/skaffold/pull/662)
* docs: add weekly meeting snippet [#675](https://github.com/GoogleContainerTools/skaffold/pull/675)
* Add labels to all k8s objects deployed by skaffold [#644](https://github.com/GoogleContainerTools/skaffold/pull/644)
* Implement packaging for helm deployment [#682](https://github.com/GoogleContainerTools/skaffold/pull/682)
* mv tagPolicy:env example [#697](https://github.com/GoogleContainerTools/skaffold/pull/697)
* windows: add appveyor [#702](https://github.com/GoogleContainerTools/skaffold/pull/702)
* add WSL support [#694](https://github.com/GoogleContainerTools/skaffold/pull/694)
* Add labels from options [#716](https://github.com/GoogleContainerTools/skaffold/pull/716)
* Add tests for helm deployment with `packaged' option [#696](https://github.com/GoogleContainerTools/skaffold/pull/696)
* Fix issue #404 - Allow to use bazel subtarget [#689](https://github.com/GoogleContainerTools/skaffold/pull/689)
* fix: allow environment variables to be used in helm values [#707](https://github.com/GoogleContainerTools/skaffold/pull/707)
* Improve Kaniko code and ns handling [#722](https://github.com/GoogleContainerTools/skaffold/pull/722)
* Support wildcards in Dockerfiles [#712](https://github.com/GoogleContainerTools/skaffold/pull/712)

Bug Fixes

* make: fix release path [#650](https://github.com/GoogleContainerTools/skaffold/pull/650)
* Fixing the licence [#652](https://github.com/GoogleContainerTools/skaffold/pull/652)
* typo fix [#660](https://github.com/GoogleContainerTools/skaffold/pull/660)
* Ignore missing authConfigs during docker build [#664](https://github.com/GoogleContainerTools/skaffold/pull/664)
* lint fixes [#669](https://github.com/GoogleContainerTools/skaffold/pull/669)
* Fix hack/dep.sh on travisCI [#680](https://github.com/GoogleContainerTools/skaffold/pull/680)
* Use git binary or fallback to go-git [#639](https://github.com/GoogleContainerTools/skaffold/pull/639)
* Fix git detection [#683](https://github.com/GoogleContainerTools/skaffold/pull/683)
* remove extraneous space [#688](https://github.com/GoogleContainerTools/skaffold/pull/688)
* Create and apply patch when adding labels to API objects [#687](https://github.com/GoogleContainerTools/skaffold/pull/687)
* Fix issue with 100% CPU usage in logs.go. [#704](https://github.com/GoogleContainerTools/skaffold/pull/704)

Updates

* Remove fsnotify [#646](https://github.com/GoogleContainerTools/skaffold/pull/646)
* Update go-containerregistry [#651](https://github.com/GoogleContainerTools/skaffold/pull/651)
* cloudbuild: increase timeout to 20m [#658](https://github.com/GoogleContainerTools/skaffold/pull/658)
* Update docker libraries [#676](https://github.com/GoogleContainerTools/skaffold/pull/676)
* Update apimachinery and client-go to kubernetes-1.11.0-beta2 [#684](https://github.com/GoogleContainerTools/skaffold/pull/684)
* Update release_notes.sh [#710](https://github.com/GoogleContainerTools/skaffold/pull/710)
* Remove unused imports [#724](https://github.com/GoogleContainerTools/skaffold/pull/724)


# v0.7.0 Release - 06/07/2018


New Features

* cmd: add skaffold deploy [#624](https://github.com/GoogleContainerTools/skaffold/pull/624)
* Remove no-manifest code. [#640](https://github.com/GoogleContainerTools/skaffold/pull/640)
* Add an mtime file watcher. [#549](https://github.com/GoogleContainerTools/skaffold/pull/549)
* Add functionality to toggle the `--wait` flag on helm install/upgrade [#633](https://github.com/GoogleContainerTools/skaffold/pull/633)
* Add kustomize deployer [#641](https://github.com/GoogleContainerTools/skaffold/pull/641)
* Add datetime tagger tagpolicy [#621](https://github.com/GoogleContainerTools/skaffold/pull/621)
* Helm: add option to generate override values.yaml based on data passed into skaffold.yaml [#632](https://github.com/GoogleContainerTools/skaffold/pull/632)
* add `--output` and `--quiet` to `skaffold build` [#606](https://github.com/GoogleContainerTools/skaffold/pull/606)
* Add the ability to express the release name as a template [#602](https://github.com/GoogleContainerTools/skaffold/pull/602)
* Simpler code that logs containers [#612](https://github.com/GoogleContainerTools/skaffold/pull/612)

Bug Fixes

* Fix image parsing in skaffold deploy [#638](https://github.com/GoogleContainerTools/skaffold/pull/638)
* Fix flaky test [#594](https://github.com/GoogleContainerTools/skaffold/pull/594)
* fix: allow an environment variable to default the deploy namespace [#497](https://github.com/GoogleContainerTools/skaffold/pull/497)
* Add BUILD and WORKSPACE files to dependencies [#636](https://github.com/GoogleContainerTools/skaffold/pull/636)
* Misc fixes to dev mode [#589](https://github.com/GoogleContainerTools/skaffold/pull/589)


Updates

* Quick Start GKE Doc - reference change for k8s-pod deployment [#615](https://github.com/GoogleContainerTools/skaffold/pull/615)
* kaniko: pin image version to v0.1.0 [#592](https://github.com/GoogleContainerTools/skaffold/pull/592)
* Refactor the envTemplate code to make it reusable [#601](https://github.com/GoogleContainerTools/skaffold/pull/601)
* Simplify runner test [#609](https://github.com/GoogleContainerTools/skaffold/pull/609)
* Move kubernetes client creation to kubernetes package [#608](https://github.com/GoogleContainerTools/skaffold/pull/608)
* Remove unused field. [#616](https://github.com/GoogleContainerTools/skaffold/pull/616)
* Remove annoying testdata folder [#614](https://github.com/GoogleContainerTools/skaffold/pull/614)
* Dockerfile should always be sent to daemon [#605](https://github.com/GoogleContainerTools/skaffold/pull/605)
* Simplify code that resolves dependencies [#610](https://github.com/GoogleContainerTools/skaffold/pull/610)
* Switch boilerplate to The Skaffold Authors. [#626](https://github.com/GoogleContainerTools/skaffold/pull/626)
* Improve runner code [#645](https://github.com/GoogleContainerTools/skaffold/pull/645)
* Simplify helm_test [#607](https://github.com/GoogleContainerTools/skaffold/pull/607)
* Replace gometalinter with GolangCI-Lint [#619](https://github.com/GoogleContainerTools/skaffold/pull/619)
* Update go-git to v4.4.0 [#634](https://github.com/GoogleContainerTools/skaffold/pull/634)
* Remove afero [#613](https://github.com/GoogleContainerTools/skaffold/pull/613)


https://github.com/GoogleContainerTools/skaffold/compare/v0.6.1...v0.7.0

# v0.6.1 Release - 5/24/2018
New Features
* Stricter YAML parsing [#570](https://github.com/GoogleContainerTools/skaffold/pull/570)
* Show helm's output and errors [#576](https://github.com/GoogleContainerTools/skaffold/pull/576)
* Support ~ in secret path for Kaniko [#455](https://github.com/GoogleContainerTools/skaffold/pull/455)
* `skaffold dev` now stops on non-build errors [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)

Bug Fixes
* GCB Build fixed [#579](https://github.com/GoogleContainerTools/skaffold/pull/579)
* Show errors of kubectl and helm commands [#560](https://github.com/GoogleContainerTools/skaffold/pull/560)
* Can now run `skaffold build` without a kubernetes cluster [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)

Updates
* Updated google/go-containerregistry [#571](https://github.com/GoogleContainerTools/skaffold/pull/571)
* Added a user agent to GCB calls [#582](https://github.com/GoogleContainerTools/skaffold/pull/582)
* Simplified runner code [#540](https://github.com/GoogleContainerTools/skaffold/pull/540)
* Silenced usage text on errors [#449](https://github.com/GoogleContainerTools/skaffold/pull/449)
* Skipped fully qualified names when replacing image names [#566](https://github.com/GoogleContainerTools/skaffold/pull/566)
* Improved docker dependencies code [#466](https://github.com/GoogleContainerTools/skaffold/pull/466)

https://github.com/GoogleContainerTools/skaffold/compare/v0.6.0...v0.6.1

# v0.6.0 Release - 5/16/2018
New Features
* Improve the `docker load` output in Bazel build [#475](https://github.com/GoogleContainerTools/skaffold/pull/475)
* `envTemplate` now supports `DIGEST_ALGO` and `DIGEST_HEX` variables [#495](https://github.com/GoogleContainerTools/skaffold/pull/495)
* Perform cleanup on `SIGPIPE` signal [#515](https://github.com/GoogleContainerTools/skaffold/pull/515)
* Learnt a `skaffold build` command [#476](https://github.com/GoogleContainerTools/skaffold/pull/476), [#553](https://github.com/GoogleContainerTools/skaffold/pull/553)
* Git tagger should use tags over commits [#552](https://github.com/GoogleContainerTools/skaffold/pull/552)

Bug Fixes
* Fixed the microservices example [#451](https://github.com/GoogleContainerTools/skaffold/pull/451)
* Don't fail if `~/.docker/config.json` doesn't exist [#454](https://github.com/GoogleContainerTools/skaffold/pull/454)
* Fix the Git Tagger name [#473](https://github.com/GoogleContainerTools/skaffold/pull/473)
* Git Tagger now handles deleted files without failing [#471](https://github.com/GoogleContainerTools/skaffold/pull/471)
* Add files to the context tarball with Unix separators [#489](https://github.com/GoogleContainerTools/skaffold/pull/489)
* Fix and improve `annotated-skaffold.yaml` [#467](https://github.com/GoogleContainerTools/skaffold/pull/467), [#520](https://github.com/GoogleContainerTools/skaffold/pull/520), [#536](https://github.com/GoogleContainerTools/skaffold/pull/536), [#542](https://github.com/GoogleContainerTools/skaffold/pull/542)
* Handle private docker registries with explicit port numbers [#525](https://github.com/GoogleContainerTools/skaffold/pull/525)
* Ignore empty manifests [#538](https://github.com/GoogleContainerTools/skaffold/pull/538)
* Default values are set after a profile is applied [#533](https://github.com/GoogleContainerTools/skaffold/pull/533)
* Remove warning when building images [#548](https://github.com/GoogleContainerTools/skaffold/pull/548)
* Some logs where not printed [#513](https://github.com/GoogleContainerTools/skaffold/pull/513)

Updates
* Improvements to the documentation [#452](https://github.com/GoogleContainerTools/skaffold/pull/452), [#453](https://github.com/GoogleContainerTools/skaffold/pull/453), [#556](https://github.com/GoogleContainerTools/skaffold/pull/556)
* Improve `kubectl` and `helm` commands output [#534](https://github.com/GoogleContainerTools/skaffold/pull/534)
* Code improvements [#485](https://github.com/GoogleContainerTools/skaffold/pull/485), [#537](https://github.com/GoogleContainerTools/skaffold/pull/537), [#544](https://github.com/GoogleContainerTools/skaffold/pull/544), [#545](https://github.com/GoogleContainerTools/skaffold/pull/545)
* Improved Git Issue template [#532](https://github.com/GoogleContainerTools/skaffold/pull/532)

https://github.com/GoogleContainerTools/skaffold/compare/v0.5.0...v0.6.0

# v0.5.0 Release - 4/23/2018
New Features
* Added kaniko builder
* Added support for "remote-manifests" in kubectl deployer
* `skaffold dev` now performs a cleanup of deployed resources on exit
* `skaffold dev` redeploys when deploy dependencies are changed (only kubectl deployer currently)

Bug Fixes
* GCB builder now uses tags correctly
* Supports multi-stage dockerfiles with onbuild commands
* Better error messages
* Fixed tagger working directory

Updates
* Switched from containers/image to google/go-containerregistry
* Integration tests now run in separate namespaces

# v0.4.0 Release - 4/12/2018
New Features
* Added `skaffold fix` command to migrate configs from v1alpha1 to v1alpha2
* Added `skaffold completion` command to output bash completion for skaffold subcommands
* Warns when an image is built but not used
* Artifacts can now be built with bazel
* Environment variable template tagger
* Support multiple document YAML files
* Helm deployer now accepts extra set values

Bug Fixes
* Logs use relative time instead of host time, which fixes issues with clock sync on local clusters
* Removed duplicate error
* Docker build args passsed to Google Container Builder
* Fixed unreliable file detection when using IntelliJ or other IDEs
* Better handling of default values
* Fixed issue with some logs being displayed twice
* Fixed .dockerignore support

Updates
* Updated go-git package
* Refactored watch package

# v0.3.0 Release - 3/29/2018
New Features
* Logs are now colored by image deployment, different container instances will get different colors in `skaffold dev`
* Better and less error prone logging
* All logs are shown for a pod with deployed containers
* Helm deployer now supports chart versions
* Helm deployer now supports custom values file path
* Logs are now muted during the build and deploy cycle
* Integration tests added
* Conditional rebuilding of changed artifacts
* Builds only triggered after a quiet period
* Duration of build and deploy is now logged


Bug Fixes
* .dockerignore now works if context is parent directory
* removed duplicate logs
* private registry authentication issues are fixed
* no logs are missed

Breaking config changes
* tagPolicy is now a struct instead of a string, to convert to the new config format

```
tagPolicy: gitCommit
```
becomes
```
tagPolicy:
    gitCommit: {}
```

* kubectl deployer no longer needs templated keys in manifests. Simply just make sure the artifacts in your skaffold.yaml correspond to the images in your kubernetes manifests and they will be automatically updated.

# v0.2.0 Release - 3/9/2018

New Features
* Added "skip-push" optimization for Docker for Desktop Kubernetes Clusters
* Examples should now be ran from their own directory
* Fixed kubernetes context for build and deploy
* Added options for GCR auth
* Set default log level to warn
* Change git commit to use short ID instead
* Helm deployer now acceptes namespace and values file
* Local builder now accepts docker build-args
* Added --tag flag for skaffold run
* Cache image configs by name
* Kubectl Generate a basic manifest if none provided

Bug fixes
* Dockerfile parsing for remote ADD file works correctly now
* Closed image config file descriptor

# v0.1.0 Release - 3/2/2018

* Added `skaffold run` command
* Added `skaffold dev` command
* Added `skaffold version` command
* Added `skaffold docker deps` command to parse dockerfile dependencies
* Added `skaffold docker context` command to generate minimal docker context tar
* Added Builders: Local Docker, Google Cloud Builder
* Added Deployers: Kubectl, Helm
* Filesystem watcher (kqueue)
* Log streaming of deploy resources
* Minikube optimizations
* Dockerfile introspection
* Added initial skaffold docker image with dependencies
* Globbing filepath config fields
* Added skaffold config
* Added initial integration test

