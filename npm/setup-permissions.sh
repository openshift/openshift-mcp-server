#!/bin/bash

echo "Setting up permissions for OpenShift MCP Server..."
echo ""
echo "Choose an option:"
echo "1. Grant full cluster admin access (recommended for full control)"
echo "2. Grant OpenShift AI specific permissions only"
echo ""
read -p "Enter your choice (1 or 2): " choice

case $choice in
  1)
    echo "Applying full cluster admin permissions..."
    oc apply -f rbac-full-cluster-admin.yaml
    echo "✅ Full cluster admin permissions granted to mcp-bot service account"
    ;;
  2)
    echo "Applying OpenShift AI specific permissions..."
    oc apply -f rbac-openshift-ai.yaml
    echo "✅ OpenShift AI permissions granted to mcp-bot service account"
    ;;
  *)
    echo "Invalid choice. Please run the script again and choose 1 or 2."
    exit 1
    ;;
esac

echo ""
echo "Verifying permissions..."
oc auth can-i '*' '*' --as=system:serviceaccount:macayaven-dev:mcp-bot

if [ $? -eq 0 ]; then
    echo "✅ Service account now has full cluster access"
else
    echo "⚠️  Service account has limited access (this is expected for Option 2)"
fi

echo ""
echo "Setup complete! Your MCP server can now manage OpenShift resources."