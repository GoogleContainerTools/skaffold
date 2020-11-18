
v1.9.0 / 2020-08-18
===================

  * add `WithDuration()` method to record a duration as milliseconds
  * add: ignore nil errors in `WithError()`
  * change trace duration to milliseconds (arguably a breaking change)

v1.8.0 / 2020-08-05
===================

  * refactor apexlogs handler to not make the AddEvents() call if there are no events to flush

v1.7.1 / 2020-08-05
===================

  * fix potential nil panic in apexlogs handler

v1.7.0 / 2020-08-03
===================

  * add FlushSync() to apexlogs handler

v1.6.0 / 2020-07-13
===================

  * update apex/logs dep to v1.0.0
  * docs: mention that Flush() is non-blocking now, use Close()

v1.5.0 / 2020-07-11
===================

  * add buffering to Apex Logs handler

v1.4.0 / 2020-06-16
===================

  * add AuthToken to apexlogs handler

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
