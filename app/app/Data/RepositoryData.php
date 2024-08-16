<?php
namespace App\Data;

use Illuminate\Support\Collection;
use Spatie\LaravelData\Data;

class RepositoryData extends Data
{
    public function __construct(
        public string $name,
        /** @var Collection<string, RepositoryFolderData> */
        public Collection $folders,
    ) {
    }
}
