function upload_ppa {
    echo "> Uploading PPA..."
    dput "ppa:cncf-buildpacks/pack-cli" ./../*.changes
}