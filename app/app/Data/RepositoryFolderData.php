<?php

namespace App\Data;

use Illuminate\Support\Collection;
use Spatie\LaravelData\Data;

class RepositoryFolderData extends Data
{
    public function __construct(
        public string $name,
        /** @var Collection<int, TestSuiteData> */
        public Collection $suites,
    ) {
    }
}
