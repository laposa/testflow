<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('test_session_service_runs', function (Blueprint $table) {
            $table->enum('type', ['automated', 'manual'])->default('automated');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('test_session_service_runs', function (Blueprint $table) {
            $table->dropColumn('type');
        });
    }
};
