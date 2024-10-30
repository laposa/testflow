<?php

namespace App\Data;

use Illuminate\Support\Collection;
use Spatie\LaravelData\Data;

class ManualTestData extends Data
{
    public string $name;
    public string $suite;
    public string $description;
    /** @var \App\Data\ManualTestStepData[] */
    public array $steps;
}
