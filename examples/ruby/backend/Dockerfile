ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ruby:2.7

WORKDIR /app
ADD Gemfile* ./
RUN bundle install

ADD . ./

CMD ["bundle","exec","puma"]
