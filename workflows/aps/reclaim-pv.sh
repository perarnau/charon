#!/bin/bash

echo "🔄 Reclaiming Released PersistentVolumes for reuse..."

# Check if PVs exist
PVS=$(kubectl get pv -o name | grep numaflow-pv || true)
if [ -z "$PVS" ]; then
    echo "❌ No numaflow PVs found. Make sure the StorageClass has been applied first."
    echo "Run: ./apply-storage-class.sh"
    exit 1
fi

echo "📋 Current PV status:"
kubectl get pv | grep numaflow-pv

echo ""
echo "🔧 Patching PVs to remove claimRef and make them available..."

# Array of PV names
PV_NAMES=("numaflow-pv-0" "numaflow-pv-1" "numaflow-pv-2")

for pv in "${PV_NAMES[@]}"; do
    # Check if PV exists
    if kubectl get pv "$pv" &>/dev/null; then
        echo "  Patching $pv..."
        kubectl patch pv "$pv" -p '{"spec":{"claimRef":null}}' 2>/dev/null || echo "    Warning: Failed to patch $pv (may already be available)"
    else
        echo "  Warning: PV $pv not found, skipping..."
    fi
done

echo ""
echo "✅ PV patching completed!"
echo ""
echo "📊 Updated PV status:"
kubectl get pv | grep numaflow-pv

echo ""
echo "💡 Next steps:"
echo "   1. Apply the JetStream configuration: kubectl apply -f workflows/aps/jetstream.yaml"
echo "   2. Check pod status: kubectl get pods | grep isbsvc"
echo "   3. If pods are still pending, wait a few seconds and check again"
