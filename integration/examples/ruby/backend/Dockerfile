FROM ruby:2.7

WORKDIR /app
ADD Gemfile* /app/
RUN bundle install

ADD . /app/

CMD ["bundle","exec","puma"]
