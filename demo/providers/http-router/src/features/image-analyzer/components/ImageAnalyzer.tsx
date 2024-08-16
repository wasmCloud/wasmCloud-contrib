import {Card} from '@/components/ui/Card';
import {Heading} from '@/components/ui/Heading';
import {Upload} from '@/features/image-analyzer/components/Upload';
import {useApi} from '@/services/backend/hooks/useApi';
import React from 'react';

function ImageAnalyzer() {
  const {analyze} = useApi();
  const handleDrop = React.useCallback(
    (file: File) => {
      analyze(file)
        .then((res) => {
          console.log('Job ID:', res.data.jobId);
        })
        .catch((err) => {
          console.error('Failed to analyze image:', err);
        });
    },
    [analyze],
  );

  return (
    <Card>
      <Heading as="h2">Upload an animal image</Heading>
      <div className="my-4">
        <Upload onDrop={handleDrop} />
      </div>
    </Card>
  );
}

export {ImageAnalyzer};
