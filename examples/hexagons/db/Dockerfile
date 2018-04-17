FROM postgres:10.3-alpine

ENV POSTGRES_PASSWORD notsosecret
ENV POSTGRES_USER hexagons
ENV POSTGRES_DB hexagons

COPY init.sql /docker-entrypoint-initdb.d/
