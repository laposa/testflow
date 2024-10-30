<?php

namespace App\Data;

use Illuminate\Support\Collection;
use Spatie\LaravelData\Data;

class ManualTestStepData extends Data
{
    public array $action = [];
    public array $input = [];
    public array $result = [];

}
