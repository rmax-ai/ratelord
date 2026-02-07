import { useMemo, useRef, useState, useEffect } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { GraphData } from '../../lib/api';

interface GraphViewProps {
  data: GraphData;
}

export default function GraphView({ data }: GraphViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });

  useEffect(() => {
    const updateDimensions = () => {
      if (containerRef.current) {
        setDimensions({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight
        });
      }
    };

    window.addEventListener('resize', updateDimensions);
    updateDimensions();

    return () => window.removeEventListener('resize', updateDimensions);
  }, []);

  const graphData = useMemo(() => {
    if (!data || !data.nodes) return { nodes: [], links: [] };
    
    const nodes = Object.values(data.nodes).map(n => ({
      id: n.id,
      group: n.type,
      label: n.label || n.id,
      val: n.type === 'identity' ? 2 : 1, // Size
      ...n
    }));
    
    const links = data.edges.map(e => ({
      source: e.from_id,
      target: e.to_id,
      type: e.type
    }));
    
    return { nodes, links };
  }, [data]);

  return (
    <div ref={containerRef} className="h-full w-full border rounded-lg overflow-hidden bg-gray-900 relative">
      <div className="absolute top-2 left-2 z-10 bg-black/50 p-2 rounded text-xs text-white">
        Nodes: {graphData.nodes.length}, Edges: {graphData.links.length}
      </div>
      <ForceGraph2D
        graphData={graphData}
        width={dimensions.width}
        height={dimensions.height}
        nodeAutoColorBy="group"
        nodeLabel="label"
        linkLabel="type"
        linkDirectionalArrowLength={3.5}
        linkDirectionalArrowRelPos={1}
        linkCurvature={0.25}
        backgroundColor="#111827" // gray-900
      />
    </div>
  );
}
