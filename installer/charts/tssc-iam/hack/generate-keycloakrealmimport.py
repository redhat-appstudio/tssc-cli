"""
Prepares a Keycloak realm export for import into the KeycloakRealmImport CRD, by
cleaning up the JSON export file and converting it to a Kubernetes YAML resource.
"""

import json
import yaml
import argparse

def clean_dict(d, is_client=False):
    """
    Recursively cleans up the Keycloak realm export dictionary removing items that
    will prevent the later import. It also removes sensitive information and
    replaces it with placeholders.
    """
    for key in [ \
        'id', \
        '_id', \
        'containerId', \
        'createdDate', \
        'createdTimestamp', \
        'webAuthnPolicyExtraOrigins', \
        'webAuthnPolicyPasswordlessExtraOrigins', \
        'keycloakVersion', \
        'localizationTexts']:
        d.pop(key, None)

    for key, value in d.items():
        if is_client and key == 'secret':
            d[key] = '__OVERWRITE_ME__'
        elif isinstance(value, dict):
            clean_dict(value, key == 'clients')
        elif isinstance(value, list):
            for item in value:
                if isinstance(item, dict):
                    clean_dict(item, key == 'clients')

def main():
    parser = argparse.ArgumentParser(description=
        'Generates a KeycloakRealmImport CRD from a Keycloak export file.')
    parser.add_argument('input', help='Keycloak realm export JSON file.')
    parser.add_argument('output', help='Output CRD resource YAML file.')

    args = parser.parse_args()

    with open(args.input, 'r') as json_file:
        data = json.load(json_file)

    clean_dict(data)

    cr = {
        "apiVersion": "k8s.keycloak.org/v2alpha1",
        "kind": "KeycloakRealmImport",
        "metadata": {
            "labels": {
                "app": "keycloak"
            },
            "namespace": "__OVERWRITE_ME__",
            "name": data['realm'],
        },
        "spec": {
            "keycloakCRName": "__OVERWRITE_ME__",
            "realm": data
        }
    }

    with open(args.output, 'w') as yaml_file:
        yaml.dump(cr, yaml_file)

if __name__ == '__main__':
    main()
