
v1.3.0 / 2020-05-26
===================

  * change FromContext() to always return a logger

v1.2.0 / 2020-05-26
===================

  * add log.NewContext() and log.FromContext(). Closes #78

v1.1.4 / 2020-04-22
===================

  * add apexlogs HTTPClient support

v1.1.3 / 2020-04-22
===================

  * add events len check before flushing to apexlogs handler

v1.1.2 / 2020-01-29
===================

  * refactor apexlogs handler to use github.com/apex/logs client

v1.1.1 / 2019-06-24
===================

  * add go.mod
  * add rough pass at apexlogs handler

v1.1.0 / 2018-10-11
===================

  * fix: cli handler to show non-string fields appropriately
  * fix: cli using fatih/color to better support windows
