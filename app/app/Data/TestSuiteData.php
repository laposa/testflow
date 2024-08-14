<?php
namespace App\Data;

use Illuminate\Support\Collection;
use Spatie\LaravelData\Data;

class TestSuiteData extends Data
{
    public function __construct(
        public string $name,
        /** @var Collection<int, TestData> */
        public Collection $tests,
    ) {
    }
}
