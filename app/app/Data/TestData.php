<?php

namespace App\Data;

use Spatie\LaravelData\Data;

class TestData extends Data
{
    public function __construct(
        public string $repository_id,
        public string $repository_name,
        public string $service_name,
        public string $service_url,
        public string $suite_name,
        public string $test_name,
        public string $workflow_id,
    ) {
    }
}
