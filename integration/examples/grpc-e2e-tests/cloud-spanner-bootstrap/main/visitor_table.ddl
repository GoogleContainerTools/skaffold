CREATE TABLE VisitorCounter (
  UserName STRING(1024) NOT NULL,
  VisitCount INT64 NOT NULL,
) PRIMARY KEY(UserName);