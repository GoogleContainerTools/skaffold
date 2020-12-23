# See https://aka.ms/containerfastmode to understand how Visual Studio uses this Dockerfile
# to build your images for faster debugging.

FROM mcr.microsoft.com/dotnet/core/aspnet:3.1-buster-slim AS base
WORKDIR /app
EXPOSE 80
EXPOSE 443

FROM mcr.microsoft.com/dotnet/core/sdk:3.1-buster AS build
COPY ["src/HelloWorld/HelloWorld.csproj", "src/HelloWorld/"]
RUN dotnet restore "src/HelloWorld/HelloWorld.csproj"
COPY . .
WORKDIR "/src/HelloWorld"
RUN ls -al 
RUN dotnet build "HelloWorld.csproj" --configuration Debug -o /app/build

FROM build AS publish
RUN dotnet publish "HelloWorld.csproj" --configuration Debug -o /app/publish

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .
ENTRYPOINT ["dotnet", "HelloWorld.dll"]