<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('test_session_items_test_session_runs', function (Blueprint $table) {
            $table->id();
            $table
                ->foreignId('session_item_id')
                ->constrained('test_session_items')
                ->onDelete('cascade')
                ->onUpdate('cascade');
            $table
                ->foreignId('session_run_id')
                ->constrained('test_session_runs')
                ->onDelete('cascade')
                ->onUpdate('cascade');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('test_session_items_test_session_runs');
    }
};
