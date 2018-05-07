FROM maven as builder
COPY pom.xml .
RUN mvn dependency:resolve dependency:resolve-plugins
ADD src ./src
RUN mvn package

FROM gcr.io/google-appengine/openjdk
COPY --from=builder target/spring-boot-sample-simple-2.0.1.RELEASE.jar app.jar
CMD ["java", "-jar", "app.jar"]
