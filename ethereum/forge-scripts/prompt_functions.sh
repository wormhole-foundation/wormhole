# prompt_functions.sh

# Function to check if an environment variable is set, and if not, prompt the user to set it
prompt_for_variable() {
    local var_name="$1"
    local custom_message="$2"

    if [ -z "${!var_name}" ]; then
        echo "$var_name is not set."
        displayed_message="Please enter the value for $var_name"
        if [ -n "$custom_message" ]; then
            displayed_message="$custom_message"
        fi
        read -p "$displayed_message: " var_value
        export "$var_name=$var_value"
        echo "$var_name set to: ${!var_name}"
    else
        echo "$var_name is already set to: ${!var_name}"
    fi
}