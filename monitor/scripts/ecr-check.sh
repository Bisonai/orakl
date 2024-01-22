#!/bin/bash

# Dependencies: curl, jq, yq

# Check the number of tags for a repository
check_repository_tags() {
    repository_name="$1"
    service_name="$2"

    # Make the API call
    response=$(curl -s -X POST \
                    -H "Content-Type: application/json" \
                    -d "{\"registryAliasName\": \"bisonai\",\"repositoryName\": \"${repository_name}\"}" \
                    https://api.us-east-1.gallery.ecr.aws/describeImageTags)

    # Count the number of tags
    available_tags=$(echo "$response" | jq '.imageTagDetails |= sort_by(.createdAt)')
    tag_count=$(echo "$available_tags" | jq '.imageTagDetails[].imageTag' | wc -l | tr -d '[:space:]')

    # # Check if the tag count exceeds the maximum allowed
    if [ "${tag_count}" -gt "${orakl_max_tag_count}" ]; then
        echo -e "$repository_name: count FAIL (${tag_count} tags)"
    else
        echo -e "$repository_name: count OK"
    fi

    # cypress
    check_used_tags "$available_tags" "cypress"
    check_used_tags "$available_tags" "baobab"
}

check_used_tags() {
    available_tags="$1"
    chain="$2"

    helm_chart=$(curl -s https://raw.githubusercontent.com/Bisonai/orakl-helm-charts/gcp-${chain}-prod/${service_name}/values.yaml)
    tag=$(echo ${helm_chart} | yq eval '.global.image.tag')
    # listener_tag=$(echo ${helm_chart} | yq eval '.global.image.listenerTag')
    # worker_tag=$(echo ${helm_chart} | yq eval '.global.image.workerTag')
    # reporter_tag=$(echo ${helm_chart} | yq eval '.global.image.reporterTag')

    cypress_found_tag=false
    for i in $(seq 0 $((tag_count - 1))); do
        cur_image_tag=$(echo "$available_tags" | jq -r ".imageTagDetails[$i].imageTag")
        if [ "$cur_image_tag" = "$tag" ]; then
            cypress_found_tag=true
        fi
    done

    if [ "$cypress_found_tag" = false ]; then
        echo "${repository_name}: ${chain} image FAIL (not found)"
    else
        echo "${repository_name}: ${chain} $tag"
    fi
}

# Set the maximum tag count
orakl_max_tag_count=5

# orakl-general has quite mixed set of tags, so we skip it
# check_repository_tags "orakl-general"
check_repository_tags "orakl-core" "core"
check_repository_tags "orakl-api" "api"
check_repository_tags "orakl-cli" "cli"
check_repository_tags "orakl-fetcher" "fetcher"
check_repository_tags "orakl-delegator" "delegator"
# check_repository_tags "orakl-goapi"
